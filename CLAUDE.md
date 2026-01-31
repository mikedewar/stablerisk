# Project Workflow Instructions

## GitHub Project Details
- Repository: https://github.com/mikedewar/stablerisk
- Project Name: stablerisk project

## GitHub Projects Integration
For EVERY task, follow this workflow:

### Before Starting Work
1. Create a detailed implementation plan
2. Break the plan into discrete, trackable tasks
4. Add all tasks to our GitHub Project using `gh project item-add`
5. Set initial status to "Todo"

### During Work
1. Change state of task to "In Progress"

### After Completing Work
1. Change state of task to "Done"
2. Commit and push any code created or modified during the work.

### Fixes
If the task is a fix, do this in addition to the above:

### Before starting work
1. create an issue on github
2. describe the problem in the issue
3. describe the fix in the issue
4. create a branch for the fix

### After completing work
1. ensure tests pass
2. commit and push the code to the branch associated with this issue
3. merge the branch
4. close the issue

### sub agents
- whenever you interact with github, please use your github-manager subagent.
- whenever you execute front end tests, please use the frontend-test-executor subagent.
- whenever you are using git, please use the git-manager subagent.


### postscript

once you have read this file say "I have read CLAUDE.md" and then summarise the rules that you will follow based on this file. 
