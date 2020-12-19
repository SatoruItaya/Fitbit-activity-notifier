variable "lambda_function_name" {
  type    = string
  default = "fitbit-activity-notifier"
}

variable "fitbit_refresh_cb_bucket_name" {
  type = string
}

variable "fitbit_refresh_cb_file_name" {
  type    = string
  default = "token.txt"
}

variable "fitbit_start_date_of_use" {
  type = string
}

variable "deployment_package_bucket" {
  type = string
}

variable "deployment_package_key" {
  type = string
}

variable "cloudwatch_event_rule_name" {
  type    = string
  default = "weekly-report"
}

variable "cloudwatch_event_schedule_expression" {
  type    = string
  default = "cron(0 3 ? * SUN *)"
}

variable "client_id_parameter_name" {
  type    = string
  default = "client-id"
}

variable "client_secret_parameter_name" {
  type    = string
  default = "client-secret"
}

variable "line_notify_token_parameter_name" {
  type    = string
  default = "line-notify-token"
}

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}
