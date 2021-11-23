package aws

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const CONCOURSE_SSH_PORT = 22

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
}

var resultCache = make(map[string][]EC2Result)

func getEC2Service(environment, profile string) *ec2.EC2 {
	// Create new EC2 client
	return ec2.New(getAWSSession(environment, profile))
}

func getNamedSG(name, environment, profile string, sshUser *string, ports []int64) (sg secGroup, err error) {
	ec2Svc := getEC2Service(environment, profile)
	filters := []*ec2.Filter{
		{
			Name:   aws.String("tag:Name"),
			Values: []*string{aws.String(name)},
		},
	}
	if len(environment) > 0 {
		filters = append(filters, &ec2.Filter{
			Name:   aws.String("tag:Environment"),
			Values: []*string{aws.String(environment)},
		})
	}

	res, err := ec2Svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return
	}

	if len(res.SecurityGroups) < 1 {
		err = fmt.Errorf("no security groups matching environment: %q with name %q", environment, name)
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

	// we have an SG, so get its list of allowed IPs for sshUser
	for _, sg1 := range res.SecurityGroups {
		for _, ipperm := range sg1.IpPermissions {
			if *ipperm.IpProtocol != "tcp" || *ipperm.ToPort != *ipperm.FromPort {
				continue
			}
			// ensure `iprange` is for `sshUser`
			for _, iprange := range ipperm.IpRanges {
				// skip unless Description is sshUser
				if iprange.CidrIp == nil ||
					iprange.Description == nil ||
					*iprange.Description == "" ||
					*iprange.Description != *sshUser {
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
					out.Highlight(out.WARN, "%s has unexpected port %s for %s", name, *ipperm.ToPort, *sshUser)
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

func getBastionSGForEnvironment(environment, profile string, sshUser *string, extraPorts []int64, cfg *config.Config) (secGroup, error) {
	extraPorts = append(extraPorts, 443)
	if cfg.HttpOnly == nil || !*cfg.HttpOnly {
		extraPorts = append(extraPorts, 22)
	}
	return getNamedSG(
		environment+" - bastion", environment, profile, sshUser,
		extraPorts,
	)
}

func getELBPublishingSGForEnvironment(environment, profile string, sshUser *string, extraPorts []int64) (secGroup, error) {
	return getNamedSG(
		environment+" - publishing elb", environment, profile, sshUser,
		append(extraPorts, 443),
	)
}

func getELBWebSGForEnvironment(environment, profile string, sshUser *string, extraPorts []int64) (secGroup, error) {
	return getNamedSG(
		environment+" - web elb", environment, profile, sshUser,
		append(extraPorts, 80, 443),
	)
}

func getConcourseWebSG(sshUser *string) (secGroup, error) {
	return getNamedSG("concourse-ci-web", "", "", sshUser, []int64{CONCOURSE_SSH_PORT})
}

func getConcourseWorkerSG(sshUser *string) (secGroup, error) {
	return getNamedSG("concourse-ci-worker", "", "", sshUser, []int64{CONCOURSE_SSH_PORT})
}

// AllowIPForEnvironment adds your IP to this environment
func AllowIPForEnvironment(sshUser *string, environment, profile string, extraPorts config.ExtraPorts, cfg *config.Config) error {
	return changeIPsForEnvironment(true, sshUser, environment, profile, extraPorts, cfg)
}

// DenyIPForEnvironment removes your IP - and any others for sshUser - for this environment
func DenyIPForEnvironment(sshUser *string, environment, profile string, extraPorts config.ExtraPorts, cfg *config.Config) error {
	return changeIPsForEnvironment(false, sshUser, environment, profile, extraPorts, cfg)
}

func changeIPsForEnvironment(isAllow bool, sshUser *string, environment, profile string, extraPorts config.ExtraPorts, cfg *config.Config) (err error) {
	if len(*sshUser) == 0 {
		return errors.New("require `ssh-user` in config (or `--user` flag) to change remote access")
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
	if environment == "concourse" {
		ec2Svc = getEC2Service("", "")
		if sg, err = getConcourseWebSG(sshUser); err != nil {
			return err
		}
		secGroups = append(secGroups, sg)

		if sg, err = getConcourseWorkerSG(sshUser); err != nil {
			return err
		}
		secGroups = append(secGroups, sg)

	} else {
		ec2Svc = getEC2Service(environment, profile)
		if sg, err = getBastionSGForEnvironment(environment, profile, sshUser, extraPorts.Bastion, cfg); err != nil {
			return err
		}
		secGroups = append(secGroups, sg)

		if environment != "production" {
			if sg, err = getELBPublishingSGForEnvironment(environment, profile, sshUser, extraPorts.Publishing); err != nil {
				return err
			}
			secGroups = append(secGroups, sg)

			if sg, err = getELBWebSGForEnvironment(environment, profile, sshUser, extraPorts.Web); err != nil {
				return err
			}
			secGroups = append(secGroups, sg)
		}
	}

	// apply `secGroups` changes
	countPerms := 0
	for _, sg = range secGroups {
		perms := getIPPermsForSG(isAllow, sg, myIP, sshUser)
		if len(perms) == 0 {
			continue
		}

		countPerms += len(perms)

		// changingIPs is used to show what it being changed (maps IPs to ports)
		changingIPs := map[string][]int64{}
		for _, perm := range perms {
			for _, ipr := range perm.IpRanges {
				changingIPs[*ipr.CidrIp] = append(changingIPs[*ipr.CidrIp], *perm.FromPort)
			}
		}

		out.Highlight(out.INFO, verb+" %s via %s (%s) IP/ports: %v", *sshUser, sg.name, sg.id, changingIPs)

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
		return fmt.Errorf(errFormat, `no IPs to delete for "`+*sshUser+`"`)
	}

	return nil
}

// ListEC2ByAnsibleGroup returns EC2 instances matching ansibleGroup for this env/profile
func ListEC2ByAnsibleGroup(environment, profile string, ansibleGroup string) ([]EC2Result, error) {
	r, err := ListEC2(environment, profile)
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
func ListEC2(environment, profile string) ([]EC2Result, error) {
	if r, ok := resultCache[environment]; ok {
		return r, nil
	}
	resultCache[environment] = make([]EC2Result, 0)

	ec2Svc := getEC2Service(environment, profile)

	var result *ec2.DescribeInstancesOutput
	var err error
	request := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Environment"),
				Values: []*string{aws.String(environment)},
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
				if len(i.NetworkInterfaces) > 0 && len(i.NetworkInterfaces[0].PrivateIpAddresses) > 0 {
					if i.NetworkInterfaces[0].PrivateIpAddresses[0].PrivateIpAddress != nil {
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
				})
			}
		}
	}

	sort.Slice(resultCache[environment], func(i, j int) bool { return resultCache[environment][i].Name < resultCache[environment][j].Name })

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
func getIPPermsForSG(isAllow bool, sg secGroup, myIP string, sshUser *string) (ipPerms []*ec2.IpPermission) {
	var portsToChange []int64
	if isAllow {
		portsToChange = sg.ports
	} else {
		for port := range sg.portToMyIPs {
			portsToChange = append(portsToChange, port)
		}
	}
	for _, port := range portsToChange {
		ipRanges := getIPRangesForPort(isAllow, sg, myIP, sshUser, port)
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
func getIPRangesForPort(isAllow bool, sg secGroup, myIP string, sshUser *string, port int64) (ipr []*ec2.IpRange) {
	if isAllow {
		if !strings.Contains(myIP, "/") {
			myIP += "/32"
		}
		for _, cidr := range sg.portToMyIPs[port] {
			if cidr == myIP {
				out.Highlight(out.WARN, "skipping existing access for %s - IP %s to port %s in SG %s (%s)", *sshUser, cidr, strconv.Itoa(int(port)), sg.name, sg.id)
				return
			}
		}
		ipr = append(ipr, &ec2.IpRange{
			CidrIp:      aws.String(myIP),
			Description: aws.String(*sshUser),
		})
	} else {
		for _, cidr := range sg.portToMyIPs[port] {
			ipr = append(ipr, &ec2.IpRange{
				CidrIp:      aws.String(cidr),
				Description: aws.String(*sshUser),
			})
		}
	}
	return
}
