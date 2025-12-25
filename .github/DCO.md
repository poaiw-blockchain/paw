# Developer Certificate of Origin (DCO)

## What is DCO?

The Developer Certificate of Origin (DCO) is a lightweight way for contributors to certify that they wrote or have the right to submit the code they are contributing to the project.

By making a contribution to this project, you certify that:

1. The contribution was created in whole or in part by you and you have the right to submit it under the open source license indicated in the file; or
2. The contribution is based upon previous work that, to the best of your knowledge, is covered under an appropriate open source license and you have the right under that license to submit that work with modifications; or
3. The contribution was provided directly to you by some other person who certified (1) or (2) and you have not modified it.

## How to Sign Your Commits

To sign your commits, use the `-s` flag:

```bash
git commit -s -m "feat: your commit message"
```

This adds a sign-off line to your commit message:

```
Signed-off-by: Your Name <your.email@example.com>
```

### Configure Git

Set your name and email if not already configured:

```bash
git config user.name "Your Name"
git config user.email "your.email@example.com"
```

### Sign-off All Commits in a PR

If you forgot to sign commits, you can amend them:

```bash
# For the last commit
git commit --amend --signoff

# For multiple commits (rebase)
git rebase HEAD~N --signoff
```

## Full DCO Text

The full Developer Certificate of Origin text is available at:
https://developercertificate.org/
