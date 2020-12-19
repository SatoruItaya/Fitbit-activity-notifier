variable "lambda_function_name" {
  type        = string
  default     = "fitbit-activity-notifier"
  description = "Lambda funtion name"
}

variable "fitbit_refresh_cb_bucket_name" {
  type        = string
  description = "S3 bucket name to place a token file"
}

variable "fitbit_refresh_cb_file_name" {
  type        = string
  default     = "token.txt"
  description = "Name of token file"
}

variable "fitbit_start_date_of_use" {
  type        = string
  description = "Start date of using Fitbit(yyyy-MM-dd)"
}

variable "deployment_package_bucket" {
  type        = string
  description = "S3 bucket name for Lambda deploy resources"
}

variable "deployment_package_key" {
  type        = string
  description = "Key of deploy resources in deployment_package_bucket"
}

variable "cloudwatch_event_rule_name" {
  type        = string
  default     = "weekly-report"
  description = "Cloudwatch Event Rule name"
}

variable "cloudwatch_event_schedule_expression" {
  type        = string
  default     = "cron(0 3 ? * SUN *)"
  description = "Schedule expression for Cloudwatch Event(Cron or Rate)"
}

variable "client_id_parameter_name" {
  type        = string
  default     = "client-id"
  description = "Parameter name of Systems Manager Parameter Store for Client ID for Fitbit"
}

variable "client_secret_parameter_name" {
  type        = string
  default     = "client-secret"
  description = "Parameter name of Systems Manager Parameter Store for Client Secret for Fitbit"
}

variable "line_notify_token_parameter_name" {
  type        = string
  default     = "line-notify-token"
  description = "Parameter name of Systems Manager Parameter Store for LINE Notify token"
}

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}
