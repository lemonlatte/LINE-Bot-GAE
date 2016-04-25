# LINE Bot for Google App Engine

## Prerequisite

- [GAE SDK for Go](https://cloud.google.com/appengine/downloads#Google_App_Engine_SDK_for_Go)

## Setup

- Update the following variables in `line-bot.go` according to basic information in your developer account:
    - X-Line-ChannelID
    - X-Line-ChannelSecret
    - X-Line-Trusted-User-With-ACL

## Deployment

```
goapp deploy -application <GAE project id> app.yaml
```

## Gotcha

- Set whilelist in LINE developer
- Set Callback URL in LINE developer

## Reference

- [amis-linebot](https://github.com/miaoski/amis-linebot)
- [LineBotで愛犬](http://qiita.com/shikajiro/items/329d660f1a457676c450)



