# Fitbit-activity-notifier

The Fitbit-activity-notifier notifies you of a custom [Fitbit](https://www.fitbit.com/global/us/home) weekly report via [LINE](https://line.me/en/).
The custom report contains following items,
- Weekly report of steps
- Top records of steps in this year
- Top records of steps in lifetime.

# Necessary things
- Fitbit
    - Fitbit account, Client ID, Client secret, Access token, Refresh token(Tokens are with a time limit.)
    - You can get them at [https://dev.fitbit.com/](https://dev.fitbit.com/). 
- LINE 
    - LINE account, [LINE Notify](https://notify-bot.line.me/ja/) token
    - You can issue a token at My page.
- AWS
    - AWS account

# Overview of Architecture
- A trigger is [CloudWatch Events](https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/WhatIsCloudWatchEvents.html) and invoke [Lambda](https://aws.amazon.com/lambda/?nc1=h_ls) funtion.
- Lambda funtion hits Fitbit API, extracts data, and create a custom report. After that, it sends a request to LINE Notify.
- You can get a custom report via LINE.

# Advance preparation
- You need to store following parameters in [Systems Manager Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html),
    - Client ID for Fitbit
    - Client Secret for Fitbit
    - LINE Notify token.
- These parameters are set as Enviromment variables for Lambda.

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
- [Web API Reference](https://dev.fitbit.com/build/reference/web-api/)
- [Fitbit Web API(Swagger UI)](https://dev.fitbit.com/build/reference/web-api/explore/)
- [orcasgit/python-fitbit](https://github.com/orcasgit/python-fitbit)
