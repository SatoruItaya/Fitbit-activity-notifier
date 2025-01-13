## Fitbit-activity-notifier/go

- This is a script running at AWS Lambda.
- Functions are following,
    - Hit Fitbit API
    - Extract data
    - Create custom report
    - Hit LINE Messaging API to send requests to LINE.
- Some variables are set as environment variables.
- Before you execute Lambda funtion first, you need to place a token file on S3 bucket by your own.
- `make deply` command will deploy lambda resources and place it on S3.
