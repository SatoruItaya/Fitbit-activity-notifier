import os
import requests
import fitbit
import boto3
from ast import literal_eval
import datetime

LINE_NOTIFY_TOKEN = os.environ["LINE_NOTIFY_TOKEN"]
HEADERS = {"Authorization": "Bearer %s" % LINE_NOTIFY_TOKEN}
URL = "https://notify-api.line.me/api/notify"

CLIENT_ID = os.environ["CLIENT_ID"]
CLIENT_SECRET = os.environ["CLIENT_SECRET"]
REFRESH_CB_BUCKET_NAME = os.environ["REFRESH_CB_BUCKET_NAME"]
REFRESH_CB_FILE_NAME = os.environ["REFRESH_CB_FILE_NAME"]

s3 = boto3.resource('s3')
refresh_cb_bucket = s3.Bucket(REFRESH_CB_BUCKET_NAME)
tmp_file_name = '/tmp/token.txt'


def update_token(token):

    f = open(tmp_file_name, 'w')
    f.write(str(token))
    f.close()
    refresh_cb_bucket.upload_file(tmp_file_name, REFRESH_CB_FILE_NAME)

    return


def format_steps(steps):
    return '{:,}'.format(steps)


def lambda_handler(event, context):

    refresh_cb_bucket.download_file(REFRESH_CB_FILE_NAME, tmp_file_name)

    tokens = open(tmp_file_name).read()
    token_dict = literal_eval(tokens)
    access_token = token_dict['access_token']
    refresh_token = token_dict['refresh_token']

    authd_client = fitbit.Fitbit(CLIENT_ID, CLIENT_SECRET, access_token=access_token, refresh_token=refresh_token, refresh_cb=update_token)

    today = datetime.date.today()

    steps_data = authd_client.time_series('activities/steps', base_date=datetime.date(today.year, 1, 1), end_date=today - datetime.timedelta(days=1))

    weekly_message = '\nWeekly Report\n\n'
    weekly_steps = {}

    for i in range(7):
        date = datetime.datetime.strptime(steps_data['activities-steps'][i - 7]['dateTime'], '%Y-%m-%d').strftime('%m/%d %a')
        steps = int(steps_data['activities-steps'][i - 7]['value'])

        weekly_message += date + ' ' + format_steps(steps) + '\n'
        weekly_steps[date] = steps

    max_date_list = [kv[0] for kv in weekly_steps.items() if kv[1] == max(weekly_steps.values())]
    min_date_list = [kv[0] for kv in weekly_steps.items() if kv[1] == min(weekly_steps.values())]
    avg = round(sum(weekly_steps.values()) / 7)

    weekly_message += 'Average:' + format_steps(avg) + '\n'
    weekly_message += 'Max:' + ','.join(max_date_list) + ' ' + format_steps(weekly_steps[max_date_list[0]]) + '\n'
    weekly_message += 'Min:' + ','.join(min_date_list) + ' ' + format_steps(weekly_steps[min_date_list[0]]) + '\n'

    message = weekly_message
    data = {'message': message}
    response = requests.post(URL, headers=HEADERS, data=data)
    print(response.text)
