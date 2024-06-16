# Fitbit-activity-notifier/python

## lambda_function.py
- The lambda_function.py is a Python script running at AWS Lambda.
- Functions are following,
    - Hit Fitbit API
    - Extract data
    - Create custom report
    - Send a request to LINE Notify.
- Some enviroment variables are set by Terraform.
- Before you execute Lambda funtion first, you need to place a token file on S3 bucket by your own. This file's format are like below,

```
{'access_token': 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 
 'expires_in': 28800, 
 'refresh_token': 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 
 'scope': ['location', 'activity', 'sleep', 'heartrate', 'social', 'settings', 'nutrition', 'profile', 'weight'], 
 'token_type': 'Bearer', 
 'user_id': 'XXXXXXX', 
 'expires_at': 1606422152.6696703}
```
You can get these values after authorization of Fitbit App.
The names of S3 bucket and token file are variables of Terraform.

## Makefile
- `make deply` command will deploy lambda resources and place it on S3.
- You need to set some parametes in Makefile, especially `ARTIFACTS_BUCKET`. This is a S3 bucket name to replace deploy resources. It can be set as an environment variable or as an option with `make deploy` as below,

```
$ make deploy ARTIFACTS_BUCKET=lambda-deploy-resources-bucket
```

