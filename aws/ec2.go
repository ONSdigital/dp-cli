package aws

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const CONCOURSE_SSH_PORT = 22
const CONCOURSE_HTTP_PORT = 80
const CONCOURSE_HTTPS_PORT = 443

type secGroup struct {
	id          string
	name        string
	ports       []int64
	portToMyIPs map[int64][]string
}

// EC2Result is the information returned for an individual EC2 instance
type EC2Result struct {
	Name          string
	Environment   string
	IPAddress     string
	AnsibleGroups []string
	GroupAKA      []string
	InstanceId    string
	LaunchTime    *time.Time
}

var resultCache = make(map[string][]EC2Result)

func getEC2Service(profile string) *ec2.EC2 {
	// Create new EC2 client
	return ec2.New(getAWSSession(profile))
}

func getNamedSG(name, environment, profile string, userName *string, ports []int64, cfg *config.Config) (sg secGroup, err error) {
	ec2Svc := getEC2Service(profile)
	filters := []*ec2.Filter{
		{
			Name:   aws.String("tag:Name"),
			Values: []*string{aws.String(name)},
		},
	}
	if len(environment) > 0 {
		expectEnvTag := environment
		if cfg.IsCI(environment) {
			expectEnvTag = "ci"
		}
		filters = append(filters, &ec2.Filter{
			Name:   aws.String("tag:Environment"),
			Values: []*string{aws.String(expectEnvTag)},
		})
	}

	res, err := ec2Svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return
	}

	if len(res.SecurityGroups) < 1 {
		err = fmt.Errorf("no security groups matching environment: %q with name %q and profile %q", environment, name, profile)
		return
	}
	if len(res.SecurityGroups) > 1 {
		err = fmt.Errorf("too many security groups matching environment: %s name: %q", environment, name)
		return
	}
	if res.SecurityGroups[0].GroupId == nil {
		err = fmt.Errorf("no groupId found for security group on environment: %q name: %q", environment, name)
		return
	}

	sg.id = *res.SecurityGroups[0].GroupId
	sg.name = name
	sg.ports = ports
	sg.portToMyIPs = make(map[int64][]string)

	// we have an SG, so get its list of allowed IPs for userName
	for _, sg1 := range res.SecurityGroups {
		for _, ipperm := range sg1.IpPermissions {
			if *ipperm.IpProtocol != "tcp" || *ipperm.ToPort != *ipperm.FromPort {
				continue
			}
			// ensure `iprange` is for `userName`
			for _, iprange := range ipperm.IpRanges {
				// skip unless Description is userName
				if iprange.CidrIp == nil ||
					iprange.Description == nil ||
					*iprange.Description == "" ||
					*iprange.Description != *userName {
					continue
				}

				// see if `ipperm` is for current allowed `ports` for this SG
				isUnusualPort := true
				for _, port := range ports {
					if *ipperm.ToPort == port {
						isUnusualPort = false
						break
					}
				}
				if isUnusualPort {
					out.Highlight(out.WARN, "%s has unexpected port %s for %s", name, *ipperm.ToPort, *userName)
				}
				// add CIDR to list that is keyed on ToPort
				sg.portToMyIPs[*ipperm.ToPort] = append(
					sg.portToMyIPs[*ipperm.ToPort],
					*iprange.CidrIp,
				)
			}
		}
	}

	return
}

func getBastionSGForEnvironment(environment, profile string, userName *string, extraPorts []int64, cfg *config.Config) (secGroup, error) {
	extraPorts = append(extraPorts, 443)
	return getNamedSG(
		environment+" - bastion", environment, profile, userName,
		extraPorts, cfg,
	)
}

func getELBPublishingSGForEnvironment(environment, profile string, userName *string, extraPorts []int64, cfg *config.Config) (secGroup, error) {
	return getNamedSG(
		environment+" - publishing elb", environment, profile, userName,
		append(extraPorts, 443), cfg,
	)
}

func getELBWebSGForEnvironment(environment, profile string, userName *string, extraPorts []int64, cfg *config.Config) (secGroup, error) {
	sgName := environment + " - web elb"
	if cfg.IsNisra(environment) {
		sgName = environment + " - cantabular-ui elb"
	}
	return getNamedSG(
		sgName, environment, profile, userName,
		append(extraPorts, 80, 443), cfg,
	)
}

func getConcourseWebSG(userName *string, profile string, cfg *config.Config) (secGroup, error) {
	return getNamedSG("concourse-ci-web", "", profile, userName, []int64{CONCOURSE_SSH_PORT, CONCOURSE_HTTP_PORT, CONCOURSE_HTTPS_PORT}, cfg)
}

func getConcourseWorkerSG(userName *string, profile string, cfg *config.Config) (secGroup, error) {
	return getNamedSG("concourse-ci-worker", "", profile, userName, []int64{CONCOURSE_SSH_PORT}, cfg)
}

// AllowIPForEnvironment adds your IP to this environment
func AllowIPForEnvironment(userName *string, environment, profile string, extraPorts config.ExtraPorts, cfg *config.Config) error {
	return changeIPsForEnvironment(true, userName, environment, profile, extraPorts, cfg)
}

// DenyIPForEnvironment removes your IP - and any others for userName - for this environment
func DenyIPForEnvironment(userName *string, environment, profile string, extraPorts config.ExtraPorts, cfg *config.Config) error {
	return changeIPsForEnvironment(false, userName, environment, profile, extraPorts, cfg)
}

func changeIPsForEnvironment(isAllow bool, userName *string, environment, profile string, extraPorts config.ExtraPorts, cfg *config.Config) (err error) {
	if len(*userName) == 0 {
		return errors.New("require `user-name` in config (or `--user` flag) to change remote access")
	}

	var myIP, verb string
	if isAllow {
		verb = "allowing"
		if myIP, err = cfg.GetMyIP(); err != nil {
			return err
		}
	} else {
		verb = "denying"
	}

	// build `secGroups` (wanted changes, per relevant security group) for `environment`
	var secGroups []secGroup
	var sg secGroup
	var ec2Svc *ec2.EC2
	if cfg.IsCI(environment) {
		ec2Svc = getEC2Service(profile)
		if sg, err = getConcourseWebSG(userName, profile, cfg); err != nil {
			return err
		}
		secGroups = append(secGroups, sg)

		if sg, err = getConcourseWorkerSG(userName, profile, cfg); err != nil {
			return err
		}
		secGroups = append(secGroups, sg)

	} else if cfg.IsNisra(environment) {
		ec2Svc = getEC2Service(profile)
		if sg, err = getELBWebSGForEnvironment(environment, profile, userName, extraPorts.Web, cfg); err != nil {
			return err
		}
		secGroups = append(secGroups, sg)

	} else {
		ec2Svc = getEC2Service(profile)
		if sg, err = getBastionSGForEnvironment(environment, profile, userName, extraPorts.Bastion, cfg); err != nil {
			return err
		}
		secGroups = append(secGroups, sg)

		if !cfg.IsLive(environment) {
			if sg, err = getELBPublishingSGForEnvironment(environment, profile, userName, extraPorts.Publishing, cfg); err != nil {
				return err
			}
			secGroups = append(secGroups, sg)
		}

		if sg, err = getELBWebSGForEnvironment(environment, profile, userName, extraPorts.Web, cfg); err != nil {
			return err
		}
		secGroups = append(secGroups, sg)

	}

	// apply `secGroups` changes
	countPerms := 0
	for _, sg = range secGroups {
		perms := getIPPermsForSG(isAllow, sg, myIP, userName)
		if len(perms) == 0 {
			continue
		}

		countPerms += len(perms)

		// changingIPs is used to show what is being changed (maps IPs to ports)
		changingIPs := map[string][]int64{}
		for _, perm := range perms {
			for _, ipr := range perm.IpRanges {
				changingIPs[*ipr.CidrIp] = append(changingIPs[*ipr.CidrIp], *perm.FromPort)
			}
		}

		out.Highlight(out.INFO, verb+" %s via %s (%s) IP/ports: %v", *userName, sg.name, sg.id, changingIPs)

		if isAllow {
			_, err = ec2Svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
				GroupId:       aws.String(sg.id),
				IpPermissions: perms,
			})
			if err != nil {
				return fmt.Errorf("error adding rules to %s SG: %s: %s", environment, sg.name, err)
			}
		} else {
			_, err = ec2Svc.RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
				GroupId:       aws.String(sg.id),
				IpPermissions: perms,
			})
			if err != nil {
				return fmt.Errorf("error removing rules from %q SG: %q: %s", environment, sg.name, err)
			}
		}
	}
	if countPerms == 0 {
		errFormat := "no changes made (%s)"
		if isAllow {
			return fmt.Errorf(errFormat, "all IPs already exist in SGs")
		}
		return fmt.Errorf(errFormat, `no IPs to delete for "`+*userName+`"`)
	}

	return nil
}

// ListEC2ByAnsibleGroup returns EC2 instances matching ansibleGroup for this env/profile
func ListEC2ByAnsibleGroup(environment, profile string, ansibleGroup string, cfg *config.Config) ([]EC2Result, error) {
	r, err := ListEC2(environment, profile, cfg)
	if err != nil {
		return r, err
	}

	var res []EC2Result
	for _, i := range r {
		for _, t := range i.AnsibleGroups {
			if t == ansibleGroup {
				res = append(res, i)
				break
			}
		}
	}

	return res, nil
}

// ListEC2 returns a list of EC2 instances which match the environment name
func ListEC2(environment, profile string, cfg *config.Config) ([]EC2Result, error) {
	if r, ok := resultCache[environment]; ok {
		return r, nil
	}
	resultCache[environment] = make([]EC2Result, 0)

	ec2Svc := getEC2Service(profile)

	var result *ec2.DescribeInstancesOutput
	var err error
	expectEnvTag := environment
	if cfg.IsCI(environment) {
		expectEnvTag = "ci"
	}
	request := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Environment"),
				Values: []*string{aws.String(expectEnvTag)},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String(ec2.InstanceStateNameRunning)},
			},
		},
	}

	for {
		if result != nil {
			if result.NextToken == nil {
				break
			}
			request.SetNextToken(*result.NextToken)
		}

		if result, err = ec2Svc.DescribeInstances(request); err != nil {
			return nil, err
		}

		for _, r := range result.Reservations {
			for _, i := range r.Instances {
				var name, ansibleGroup string
				for _, tag := range i.Tags {
					if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
						name = *tag.Value
						continue
					} else if tag.Key != nil && *tag.Key == "AnsibleGroup" && tag.Value != nil {
						ansibleGroup = *tag.Value
						continue
					}
				}
				var ipAddr string
				if cfg.IsCI(environment) && cfg.IsAWSA(environment) {
					if len(i.NetworkInterfaces) > 0 &&
						i.NetworkInterfaces[0].Association != nil &&
						len(*i.NetworkInterfaces[0].Association.PublicIp) > 0 &&
						i.NetworkInterfaces[0].Association.PublicIp != nil {
						ipAddr = *i.NetworkInterfaces[0].Association.PublicIp
					}
				} else {
					if len(i.NetworkInterfaces) > 0 &&
						len(i.NetworkInterfaces[0].PrivateIpAddresses) > 0 &&
						i.NetworkInterfaces[0].PrivateIpAddresses[0].PrivateIpAddress != nil {
						ipAddr = *i.NetworkInterfaces[0].PrivateIpAddresses[0].PrivateIpAddress
					}
				}
				resultCache[environment] = append(resultCache[environment], EC2Result{
					Name:          name,
					IPAddress:     ipAddr,
					Environment:   environment,
					AnsibleGroups: strings.Split(ansibleGroup, ","),
					GroupAKA:      []string{},
					InstanceId:    *i.InstanceId,
					LaunchTime:    i.LaunchTime,
				})
			}
		}
	}

	sort.Slice(resultCache[environment], func(i, j int) bool {
		if resultCache[environment][i].Name == resultCache[environment][j].Name {
			return resultCache[environment][i].LaunchTime.Before(*resultCache[environment][j].LaunchTime)
		}
		return resultCache[environment][i].Name < resultCache[environment][j].Name
	})

	// add (e.g.) "publishing 2" to GroupAKA field, now that the list is sorted
	countGroup := make(map[string]int)
	for i := range resultCache[environment] {
		for _, grp := range resultCache[environment][i].AnsibleGroups {
			countGroup[grp]++
			resultCache[environment][i].GroupAKA = append(resultCache[environment][i].GroupAKA, fmt.Sprintf("%s %d", grp, countGroup[grp]))
		}
	}

	return resultCache[environment], nil
}

// getIPPermsForSG returns the permissions for all ports for this SG
func getIPPermsForSG(isAllow bool, sg secGroup, myIP string, userName *string) (ipPerms []*ec2.IpPermission) {
	var portsToChange []int64
	if isAllow {
		portsToChange = sg.ports
	} else {
		for port := range sg.portToMyIPs {
			portsToChange = append(portsToChange, port)
		}
	}
	for _, port := range portsToChange {
		ipRanges := getIPRangesForPort(isAllow, sg, myIP, userName, port)
		if len(ipRanges) == 0 {
			continue
		}
		ipPerms = append(ipPerms, &ec2.IpPermission{
			IpProtocol: aws.String("tcp"),
			FromPort:   aws.Int64(port),
			ToPort:     aws.Int64(port),
			IpRanges:   ipRanges,
		})
	}
	return ipPerms
}

// getIPRangesForPort returns the IPs that we will allow/deny for `port`
// Skips:
// - for `allow`: existing IPs for this SG/port
// - for `deny`:  missing IPs for this SG/port
func getIPRangesForPort(isAllow bool, sg secGroup, myIP string, userName *string, port int64) (ipr []*ec2.IpRange) {
	if isAllow {
		if !strings.Contains(myIP, "/") {
			myIP += "/32"
		}
		for _, cidr := range sg.portToMyIPs[port] {
			if cidr == myIP {
				out.Highlight(out.WARN, "skipping existing access for %s - IP %s to port %s in SG %s (%s)", *userName, cidr, strconv.Itoa(int(port)), sg.name, sg.id)
				return
			}
		}
		ipr = append(ipr, &ec2.IpRange{
			CidrIp:      aws.String(myIP),
			Description: aws.String(*userName),
		})
	} else {
		for _, cidr := range sg.portToMyIPs[port] {
			ipr = append(ipr, &ec2.IpRange{
				CidrIp:      aws.String(cidr),
				Description: aws.String(*userName),
			})
		}
	}
	return
}
