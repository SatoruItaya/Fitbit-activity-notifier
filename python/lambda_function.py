import os
import requests
import fitbit
import boto3
from ast import literal_eval
import datetime

LINE_NOTIFY_TOKEN_PARAMETER_NAME = os.environ["LINE_NOTIFY_TOKEN_PARAMETER_NAME"]
URL = "https://notify-api.line.me/api/notify"

CLIENT_ID_PARAMETER_NAME = os.environ["CLIENT_ID_PARAMETER_NAME"]
CLIENT_SECRET_PARAMETER_NAME = os.environ["CLIENT_SECRET_PARAMETER_NAME"]
REFRESH_CB_BUCKET_NAME = os.environ["REFRESH_CB_BUCKET_NAME"]
REFRESH_CB_FILE_NAME = os.environ["REFRESH_CB_FILE_NAME"]

# Fitbit API specification
LIMIT_DAYS = 1095

s3 = boto3.resource('s3')
refresh_cb_bucket = s3.Bucket(REFRESH_CB_BUCKET_NAME)
tmp_file_name = '/tmp/token.txt'

today = datetime.datetime.today()


def get_parameter(name):
    ssm_client = boto3.client('ssm')
    response = ssm_client.get_parameter(
        Name=name,
        WithDecryption=True
    )

    return response['Parameter']['Value']


def update_token(token):

    f = open(tmp_file_name, 'w')
    f.write(str(token))
    f.close()
    refresh_cb_bucket.upload_file(tmp_file_name, REFRESH_CB_FILE_NAME)

    return


def format_steps(steps):
    return '{:,}'.format(steps)


def get_lifetime_steps_data(str_start_date, authd_client):

    # create dictionary {key:date(datetime.datetime), value:step(string)}
    lifetime_steps_data_dict = {}

    # define start_date as datetime.datetime from cloudwatch event input
    start_date = datetime.datetime.strptime(str_start_date, '%Y-%m-%d')

    # Number of target days
    rest_target_days = (today - start_date).days

    count = 0

    while rest_target_days > 0:

        tmp_end_date = today - datetime.timedelta(days=1 + LIMIT_DAYS * count)

        if rest_target_days > LIMIT_DAYS:
            tmp_start_date = tmp_end_date - datetime.timedelta(days=LIMIT_DAYS - 1)
        else:
            tmp_start_date = start_date

        tmp_steps_data = authd_client.time_series('activities/steps',
                                                  base_date=datetime.datetime.strftime(tmp_start_date, '%Y-%m-%d'),
                                                  end_date=datetime.datetime.strftime(tmp_end_date, '%Y-%m-%d'))

        for i in tmp_steps_data['activities-steps']:
            lifetime_steps_data_dict[datetime.datetime.strptime(i['dateTime'], '%Y-%m-%d')] = int(i['value'])

        rest_target_days -= LIMIT_DAYS
        count += 1

    return lifetime_steps_data_dict


def create_weekly_report(steps_dict):

    two_weeks_steps_dict = {k: v for k, v in steps_dict.items() if k >= today - datetime.timedelta(days=15)}

    if len(two_weeks_steps_dict) < 14:
        return "There is not enough data."

    # The type of sorted_week_steps is list of tuple.
    sorted_week_steps = sorted(two_weeks_steps_dict.items(), key=lambda x: x[0])
    week_steps = 0
    previous_week_steps = 0

    message = 'Weekly Report\n\n'

    for i in range(7):
        date = sorted_week_steps[7 + i][0].strftime('%m/%d %a')
        steps = sorted_week_steps[7 + i][1]
        week_steps += steps
        previous_week_steps += sorted_week_steps[i][1]

        message += date + ' ' + format_steps(steps) + '\n'

    avg = round(week_steps / 7)

    one_week_steps_dict = {k: v for k, v in two_weeks_steps_dict.items() if k >= today - datetime.timedelta(days=8)}
    max_date_list = [kv[0].strftime('%m/%d') for kv in one_week_steps_dict.items()
                     if kv[1] == max(one_week_steps_dict.values())]
    min_date_list = [kv[0].strftime('%m/%d') for kv in one_week_steps_dict.items()
                     if kv[1] == min(one_week_steps_dict.values())]

    message += '\n'
    message += 'Total: ' + format_steps(week_steps) + \
        '(' + '{:+,}'.format(week_steps - previous_week_steps) + ')\n'
    message += 'Average: ' + format_steps(avg) + '\n'
    message += 'Max: ' + ','.join(max_date_list) + '\n'
    message += 'Min: ' + ','.join(min_date_list) + '\n'

    return message


def create_yearly_top_records_report(steps_dict):

    year_steps_dict = {k: v for k, v in steps_dict.items() if k >= datetime.datetime(today.year, 1, 1)}

    if len(year_steps_dict) < 5:
        return "There is not enough data."

    # The type of sorted_year_steps is list of tuple.
    sorted_year_steps = sorted(year_steps_dict.items(), key=lambda x: x[1], reverse=True)
    message = 'Top Records in This Year\n\n'

    for i in range(5):
        message += format_steps(sorted_year_steps[i][1]) + \
            '(' + sorted_year_steps[i][0].strftime('%m/%d') + ')\n'

    return message


def create_lifetime_top_records_report(steps_dict):

    if len(steps_dict) < 5:
        return "There is not enough data."

    # The type of sorted_lifetime_steps is list of tuple.
    sorted_lifetime_steps = sorted(steps_dict.items(), key=lambda x: x[1], reverse=True)
    message = 'Top Records in Lifetime\n\n'

    for i in range(5):
        message += format_steps(sorted_lifetime_steps[i][1]) + \
            '(' + sorted_lifetime_steps[i][0].strftime('%Y/%m/%d') + ')\n'

    return message


def lambda_handler(event, context):

    refresh_cb_bucket.download_file(REFRESH_CB_FILE_NAME, tmp_file_name)

    tokens = open(tmp_file_name).read()
    token_dict = literal_eval(tokens)
    access_token = token_dict['access_token']
    refresh_token = token_dict['refresh_token']

    authd_client = fitbit.Fitbit(get_parameter(CLIENT_ID_PARAMETER_NAME), get_parameter(CLIENT_SECRET_PARAMETER_NAME), access_token=access_token,
                                 refresh_token=refresh_token, refresh_cb=update_token)

    # {key:date(datetime.datetime), value:step(string)}
    lifetime_steps_data_dict = get_lifetime_steps_data(event['start_date'], authd_client)

    message = '\n'
    message += '======================\n'
    message += create_weekly_report(lifetime_steps_data_dict)
    message += '======================\n'
    message += create_yearly_top_records_report(lifetime_steps_data_dict)
    message += '======================\n'
    message += create_lifetime_top_records_report(lifetime_steps_data_dict)

    headers = {"Authorization": "Bearer %s" % get_parameter(LINE_NOTIFY_TOKEN_PARAMETER_NAME)}
    data = {'message': message}
    response = requests.post(URL, headers=headers, data=data)
    print(response.text)
