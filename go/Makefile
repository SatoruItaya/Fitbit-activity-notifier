ecr_login:
	 aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

build_and_deploy:
	@make build
	@make push
	@make deploy

build:
	docker buildx build --platform=linux/arm64 -t ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${RESOURCE_NAME}:latest .

push:
	docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${RESOURCE_NAME}:latest

deploy:
	aws lambda update-function-code --region ${AWS_REGION} --function-name ${RESOURCE_NAME} --image-uri ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${RESOURCE_NAME}:latest
