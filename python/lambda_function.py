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

today = datetime.date.today()


def update_token(token):

    f = open(tmp_file_name, 'w')
    f.write(str(token))
    f.close()
    refresh_cb_bucket.upload_file(tmp_file_name, REFRESH_CB_FILE_NAME)

    return


def format_steps(steps):
    return '{:,}'.format(steps)


def create_weekly_report(steps_dict):
    weekly_message = '\nWeekly Report\n'
    weekly_steps = {}
    previous_weekly_steps = 0

    for i in range(7):
        date = (today - datetime.timedelta(days=7 - i)).strftime('%m/%d %a')
        steps = steps_dict[(today - datetime.timedelta(days=7 - i)).strftime('%Y-%m-%d')]
        previous_weekly_steps += steps_dict[(today - datetime.timedelta(days=14 - i)).strftime('%Y-%m-%d')]

        weekly_message += date + ' ' + format_steps(steps) + ' steps\n'
        weekly_steps[date] = steps

    total = sum(weekly_steps.values())
    avg = round(total / 7)
    max_date_list = [kv[0] for kv in weekly_steps.items() if kv[1] == max(weekly_steps.values())]
    min_date_list = [kv[0] for kv in weekly_steps.items() if kv[1] == min(weekly_steps.values())]

    weekly_message += '\nTotal: ' + format_steps(total) + ' steps ' + '(' + '{:+,}'.format(total - previous_weekly_steps) + ')\n'
    weekly_message += 'Average: ' + format_steps(avg) + ' steps\n'
    weekly_message += 'Max: ' + ','.join(max_date_list) + '\n'
    weekly_message += 'Min: ' + ','.join(min_date_list) + '\n'

    return weekly_message


def create_yearly_report(yearly_steps_data):

    scores_sorted = sorted(yearly_steps_data, key=lambda x: x['value'], reverse=True)

    yearly_message = '\nTop Records in This Year\n'

    for i in range(5):
        yearly_message += format_steps(scores_sorted[i]['value']) + ' steps' \
            + '(' + datetime.datetime.strptime(scores_sorted[i]['dateTime'], '%Y-%m-%d').strftime('%m/%d') + ')\n'

    return yearly_message


def lambda_handler(event, context):

    refresh_cb_bucket.download_file(REFRESH_CB_FILE_NAME, tmp_file_name)

    tokens = open(tmp_file_name).read()
    token_dict = literal_eval(tokens)
    access_token = token_dict['access_token']
    refresh_token = token_dict['refresh_token']

    authd_client = fitbit.Fitbit(CLIENT_ID, CLIENT_SECRET, access_token=access_token, refresh_token=refresh_token, refresh_cb=update_token)

    yearly_steps_data = authd_client.time_series('activities/steps',
                                                 base_date=datetime.date(today.year, 1, 1), end_date=today - datetime.timedelta(days=1))

    for i in yearly_steps_data['activities-steps']:
        i['value'] = int(i['value'])

    # create map {key:date, value:step}
    lifetime_steps_dict = {}
    lifetime_steps_data = authd_client.time_series('activities/steps', period='max')
    for i in lifetime_steps_data['activities-steps'][:-1]:
        lifetime_steps_dict[datetime.datetime.strptime(i['dateTime'], '%Y-%m-%d').strftime('%Y-%m-%d')] = int(i['value'])

    message = ''
    message += create_weekly_report(lifetime_steps_dict)
    message += '======================'
    message += create_yearly_report(yearly_steps_data['activities-steps'])

    data = {'message': message}
    response = requests.post(URL, headers=HEADERS, data=data)
    print(response.text)
