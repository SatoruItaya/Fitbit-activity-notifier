import os
import requests
import fitbit

LINE_NOTIFY_TOKEN = os.environ["LINE_NOTIFY_TOKEN"]
HEADERS = {"Authorization": "Bearer %s" % LINE_NOTIFY_TOKEN}
URL = "https://notify-api.line.me/api/notify"

CLIENT_ID = os.environ["CLIENT_ID"]
CLIENT_SECRET = os.environ["CLIENT_SECRET"]
ACCESS_TOKEN = os.environ["ACCESS_TOKEN"]
REFRESH_TOKEN = os.environ["REFRESH_TOKEN"]


def lambda_handler(event, context):

    authd_client = fitbit.Fitbit(CLIENT_ID, CLIENT_SECRET, access_token=ACCESS_TOKEN, refresh_token=REFRESH_TOKEN)

    steps_data = authd_client.time_series('activities/steps', period='1m')
    print(steps_data)

    data = {'message': steps_data['activities-steps']}
    response = requests.post(URL, headers=HEADERS, data=data)
    print(response.text)
