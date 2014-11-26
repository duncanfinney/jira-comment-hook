jira-comment-hook
=================

A GO slackbot that polls and forwards Jira comments

###[Associated Blog Post](http://duncanfinney.github.io/2014/11/23/creating-a-slack-bot-for-jira-comments/)


Usage
=====

1. Add new slack incoming webhook: https://l11.slack.com/services/new/incoming-webhook
2. Register with heroku (free GO hosting)
3. Deploy:

```
git clone git@github.com:duncanfinney/jira-comment-hook.git
git push heroku
heroku config:set SLACK_WEBHOOK=https://hooks.slack.com/services/XXXXXXXX/XXXXXXXX/XXXXXXXXXXXXXXX
heroku config:set JIRA_URL=https://yourinstance.atlassian.net
heroku config:set JIRA_USERNAME=duncan
heroku config:set JIRA_PASSWORD=1337JiraPass
heroku ps:scale worker=1
```
