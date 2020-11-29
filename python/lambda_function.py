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

today = datetime.datetime.today()


def update_token(token):

    f = open(tmp_file_name, 'w')
    f.write(str(token))
    f.close()
    refresh_cb_bucket.upload_file(tmp_file_name, REFRESH_CB_FILE_NAME)

    return


def format_steps(steps):
    return '{:,}'.format(steps)


def create_weekly_report(steps_dict):

    two_weeks_steps_dict = {k: v for k, v in steps_dict.items() if k >= today - datetime.timedelta(days=15)}
    sorted_week_steps = sorted(two_weeks_steps_dict.items(), key=lambda x: x[0])
    week_steps = 0
    previous_week_steps = 0

    weekly_message = '\nWeekly Report\n'

    for i in range(7):
        date = sorted_week_steps[7 + i][0].strftime('%m/%d %a')
        steps = sorted_week_steps[7 + i][1]
        week_steps += steps
        previous_week_steps += sorted_week_steps[i][1]

        weekly_message += date + ' ' + format_steps(steps) + ' steps\n'

    avg = round(week_steps / 7)

    one_week_steps_dict = {k: v for k, v in two_weeks_steps_dict.items() if k >= today - datetime.timedelta(days=8)}
    max_date_list = [kv[0].strftime('%m/%d') for kv in one_week_steps_dict.items()
                     if kv[0] >= today - datetime.timedelta(days=8) and kv[1] == max(one_week_steps_dict.values())]
    min_date_list = [kv[0].strftime('%m/%d') for kv in one_week_steps_dict.items()
                     if kv[0] >= today - datetime.timedelta(days=8) and kv[1] == min(one_week_steps_dict.values())]

    weekly_message += '\n'
    weekly_message += 'Total: ' + format_steps(week_steps) + ' steps ' + \
        '(' + '{:+,}'.format(week_steps - previous_week_steps) + ')\n'
    weekly_message += 'Average: ' + format_steps(avg) + ' steps\n'
    weekly_message += 'Max: ' + ','.join(max_date_list) + '\n'
    weekly_message += 'Min: ' + ','.join(min_date_list) + '\n'

    return weekly_message


def create_yearly_top_report(steps_dict):

    yearly_steps_dict = {k: v for k, v in steps_dict.items() if k > datetime.datetime(today.year, 1, 1)}
    sorted_yearly_steps = sorted(yearly_steps_dict.items(), key=lambda x: x[1], reverse=True)
    yearly_message = '\nTop Records in This Year\n'

    for i in range(5):
        yearly_message += format_steps(sorted_yearly_steps[i][1]) + ' steps' \
            + '(' + sorted_yearly_steps[i][0].strftime('%m/%d') + ')\n'

    return yearly_message


def lambda_handler(event, context):

    refresh_cb_bucket.download_file(REFRESH_CB_FILE_NAME, tmp_file_name)

    tokens = open(tmp_file_name).read()
    token_dict = literal_eval(tokens)
    access_token = token_dict['access_token']
    refresh_token = token_dict['refresh_token']

    authd_client = fitbit.Fitbit(CLIENT_ID, CLIENT_SECRET,
                                 access_token=access_token, refresh_token=refresh_token, refresh_cb=update_token)

    yearly_steps_data = authd_client.time_series('activities/steps',
                                                 base_date=datetime.date(today.year, 1, 1), end_date=today - datetime.timedelta(days=1))

    for i in yearly_steps_data['activities-steps']:
        i['value'] = int(i['value'])

    # create map {key:date(datetime.datetime), value:step(string)}
    lifetime_steps_date_dict = {}
    lifetime_steps_data = authd_client.time_series('activities/steps', period='max')

    for i in lifetime_steps_data['activities-steps'][:-1]:
        lifetime_steps_date_dict[datetime.datetime.strptime(i['dateTime'], '%Y-%m-%d')] = int(i['value'])

    message = ''
    message += create_weekly_report(lifetime_steps_date_dict)
    message += '======================'
    message += create_yearly_top_report(lifetime_steps_date_dict)

    data = {'message': message}
    response = requests.post(URL, headers=HEADERS, data=data)
    print(response.text)
