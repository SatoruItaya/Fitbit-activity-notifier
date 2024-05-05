## Fitbit-activity-notifier/go

- This is a script running at AWS Lambda.
- Functions are following,
    - Hit Fitbit API
    - Extract data
    - Create custom report
    - Send a request to LINE Notify.
- Some variables are set as environment variables.
- Before you execute Lambda funtion first, you need to place a token file on S3 bucket by your own.