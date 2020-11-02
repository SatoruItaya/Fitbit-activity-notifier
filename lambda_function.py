import os
import requests

LINE_NOTIFY_TOKEN = os.environ["LINE_NOTIFY_TOKEN"]
HEADERS = {"Authorization": "Bearer %s" % LINE_NOTIFY_TOKEN}
URL = "https://notify-api.line.me/api/notify"

def lambda_handler(event, context):
    data = {'message': "send_test"}
    requests.post(URL, headers=HEADERS, data=data)
