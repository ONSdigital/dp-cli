## Converting 'main' branch to 'master'

### Description
 
ONSdigital is moving towards the convention of naming the previously known `master` branch to `main`. The 
repo-generation tool in this repository now generates repositories with the branches `main` and `develop`. However, 
our CI infrastructure requires a refactor for us to complete this migration of using `main`. In the interim 
branches should use `master` instead of `main`. Below should act as a guide on how to complete this.

### What needs changing

- Change branch name `main` to `master`
- Update security settings for the `master` branch to match that of `main`

### How

Ensure that you have the `main` branch checked out locally by running the following command in your terminal
```bash
git checkout main
```

Locally rename the `main` branch to `master` by running the following command in your terminal  
```bash
git branch -m main master
```

Push the new `master` branch to the remote by running the following command in your terminal
```bash
git push -u origin master
```

Remove branch protections and set the default branch in GitHub
1. Navigate to the GitHub repository `https://github.com/ONSdigital/{Repository-Name}` - where `{Repository-Name}` is replaced by the name of your repository
2. Click 'Settings' - it will have an icon of a cog next to it and is located towards the top of the page
3. Select 'Branches' from the left menu
4. Under the section 'Branch protection rules' find the entry 'main' then to the right of it click delete

Remove the branch `main` from the remote by running the following command in your terminal
```bash
git push origin --delete main
```

Finally, we need to update the `master` branches security settings
1. Navigate to the GitHub repository `https://github.com/ONSdigital/{Repository-Name}` - where `{Repository-Name}` is replaced by the name of your repository
2. Click 'Settings' - it will have an icon of a cog next to it and is located towards the top of the page
3. Select 'Branches' from the left menu
4. Next to 'Branch protection rules' click 'Add rule'
5. In the box 'Branch name pattern' type "master"
6. The next steps are all to be completed under the section 'Protect matching branches' of the page
7. Check the box for 'Require pull request reviews before merging'
8. Check the box for 'Dismiss stale pull request approvals when new commits are pushed'
9. Check the box for 'Require status checks to pass before merging'
10. Check the box for 'Require branches to be up to date before merging'
11. Check the box for 'Require signed commits'
12. Check the box for 'Include administrators'
13. Check the box for 'Restrict who can push to matching branches'
14. Under the checkbox 'Restrict who can push to matching branches' enter 'ONSdigital/digitalpublishing'
15. Double check all security settings ensuring that all the above has been done
16. Click 'Save Changes'

This task is now complete.