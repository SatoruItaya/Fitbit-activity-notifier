resource "aws_lambda_function" "lambda" {
  function_name = var.lambda_function_name
  handler       = "lambda_function.lambda_handler"
  role          = aws_iam_role.lambda_iam_role.arn

  runtime = "python3.7"

  s3_bucket = var.deployment_package_bucket
  s3_key    = var.deployment_package_key

  timeout = 60

  environment {
    variables = {
      CLIENT_ID_PARAMETER_NAME         = var.client_id_parameter_name
      CLIENT_SECRET_PARAMETER_NAME     = var.client_secret_parameter_name
      LINE_NOTIFY_TOKEN_PARAMETER_NAME = var.line_notify_token_parameter_name
      REFRESH_CB_BUCKET_NAME           = var.fitbit_refresh_cb_bucket_name
      REFRESH_CB_FILE_NAME             = var.fitbit_refresh_cb_file_name
    }
  }
}

resource "aws_cloudwatch_log_group" "lambda_log_group" {
  name              = "/aws/lambda/${var.lambda_function_name}"
  retention_in_days = 3
}

resource "aws_iam_role" "lambda_iam_role" {
  name = var.lambda_function_name

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_policy" "lambda_iam_policy" {
  name        = var.lambda_function_name
  path        = "/"
  description = "IAM policy for logging from ${var.lambda_function_name}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "${aws_cloudwatch_log_group.lambda_log_group.arn}:*",
      "Effect": "Allow"
    },
    {
      "Action": "ssm:GetParameter",
      "Resource": [
        "arn:aws:ssm:${var.aws_region}:${var.aws_account}:parameter/${var.client_id_parameter_name}",
        "arn:aws:ssm:${var.aws_region}:${var.aws_account}:parameter/${var.client_secret_parameter_name}",
        "arn:aws:ssm:${var.aws_region}:${var.aws_account}:parameter/${var.line_notify_token_parameter_name}"
      ],
      "Effect": "Allow"
    },
    {
      "Action": [
        "s3:Put*",
        "s3:Get*"
      ],
      "Resource": "${aws_s3_bucket.fitbit_refresh_cb_bucket.arn}/*",
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "lambda_iam_role_policy_attachment" {
  role       = aws_iam_role.lambda_iam_role.name
  policy_arn = aws_iam_policy.lambda_iam_policy.arn
}

resource "aws_s3_bucket" "fitbit_refresh_cb_bucket" {
  bucket        = var.fitbit_refresh_cb_bucket_name
  acl           = "private"
  force_destroy = false
}

