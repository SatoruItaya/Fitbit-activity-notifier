ARTIFACTS_DIR := artifacts
ZIP_FILE_NAME := lambda
LAMBDA_FUNCTION_NAME := fitbit-activity-notifier

deploy:
	cp lambda_function.py $(ARTIFACTS_DIR)
	pip install -r requirements.txt -t $(ARTIFACTS_DIR)
	zip $(ZIP_FILE_NAME) -r $(ARTIFACTS_DIR)
	aws lambda update-function-code --function-name $(LAMBDA_FUNCTION_NAME) --zip-file fileb://$(ZIP_FILE_NAME).zip
	rm -rf $(ZIP_FILE_NAME).zip
