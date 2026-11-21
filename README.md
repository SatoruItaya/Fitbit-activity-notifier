# Fitbit-activity-notifier

The Fitbit-activity-notifier notifies you of a custom [Fitbit](https://www.fitbit.com/global/us/home) weekly report via [LINE](https://line.me/en/).
Data is fetched from the [Google Health API](https://developers.google.com/health) (the successor of the Fitbit Web API).
The custom report contains following items,
- Weekly report of steps
- Top records of steps in this year
- Top records of steps in lifetime.

# Necessary things
- Google Health API
    - Google account linked to your Fitbit data
    - Google Cloud project with the Google Health API enabled
    - OAuth client (Client ID, Client secret) and a refresh token with the scope `https://www.googleapis.com/auth/googlehealth.activity_and_fitness.readonly`
    - See [Get a refresh token](#get-a-refresh-token).
- LINE 
    - LINE account
    - Provider, Channel, and Channel Access Token
        - You can create these resources at [LINE Developers](https://developers.line.biz/ja/).
- AWS
    - AWS account

# Overview of Architecture
- A trigger is [CloudWatch Events](https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/WhatIsCloudWatchEvents.html) and invoke [Lambda](https://aws.amazon.com/lambda/?nc1=h_ls) funtion.
- Lambda funtion hits Google Health API, extracts data, and create a custom report. After that, it sends a request to LINE Messaging API.
- You can get a custom report via LINE.

# Advance preparation
- You need to store following parameters in [Systems Manager Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html),
    - OAuth Client ID for Google Health API
    - OAuth Client Secret for Google Health API
    - LINE channel access token and user ID.
- These parameters are set as Enviromment variables for Lambda.

# Get a refresh token
1. Enable the Google Health API in your Google Cloud project, and create an OAuth client (type: Web application) with a redirect URI (e.g. `https://www.google.com`).
2. Configure Google Auth Platform.
    - Audience: Add your Google account to "Test users".
    - Data Access: Add the scope `https://www.googleapis.com/auth/googlehealth.activity_and_fitness.readonly`.
    - Branding: Set "Application home page", "Application privacy policy link" and "Authorized domains". They are required to publish the app.
    - Audience: Click "Publish app" to change the publishing status to "In production". Verification is not necessary for personal use (unverified apps are limited to 100 users).
3. Open the following URL in a browser and allow the access. `access_type=offline` and `prompt=consent` are necessary to get a refresh token.
    ```
    https://accounts.google.com/o/oauth2/v2/auth?client_id=<CLIENT_ID>&redirect_uri=https://www.google.com&response_type=code&access_type=offline&prompt=consent&scope=https://www.googleapis.com/auth/googlehealth.activity_and_fitness.readonly
    ```
4. Exchange the `code` in the redirected URL for tokens.
    ```
    curl -X POST https://oauth2.googleapis.com/token \
      -d code=<CODE> -d client_id=<CLIENT_ID> -d client_secret=<CLIENT_SECRET> \
      -d redirect_uri=https://www.google.com -d grant_type=authorization_code
    ```
5. Put the `refresh_token` in the response on the S3 bucket.

Note: While the publishing status is "Testing", refresh tokens expire in 7 days. If the token response contains `refresh_token_expires_in`, the app is still in "Testing".

# Structure
```
.
├── go
└── _python(NOT MAINTAINED)
    Python script for AWS Lamdba and Makefile to deploy
```

More details are shown in each README.

# Example of a custom report

```
======================
Weekly Report

12/06 Sun 11,440
12/07 Mon 16,258
12/08 Tue 10,739
12/09 Wed 6,998
12/10 Thu 12,204
12/11 Fri 7,413
12/12 Sat 15,030

Total: 80,082(+19,126)
Average: 11,440
Max: 12/07
Min: 12/09
======================
Top Records in This Year

27,726(11/08)
23,851(03/01)
22,477(01/26)
20,713(09/28)
20,448(09/11)
======================
Top Records in Lifetime

28,225(2017/12/03)
27,726(2020/11/08)
25,993(2018/03/17)
25,702(2018/09/23)
25,503(2018/03/11)
```

```
Running Report

Weekly Distance: 3.00km
Yearly Distance: 10.00km`
```

# References
- [Google Health API](https://developers.google.com/health)
- [Migration guide from Fitbit Web API](https://developers.google.com/health/migration)
- [orcasgit/python-fitbit](https://github.com/orcasgit/python-fitbit)
