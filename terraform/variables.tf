variable "aws_account" {
  type = string
}

variable "aws_region" {
  type    = string
  default = "ap-northeast-1"
}

variable "lambda_function_name" {
  type    = string
  default = "fitbit-activity-notifier"
}

variable "fitbit_refresh_cb_bucket_name" {
  type = string
}

variable "deployment_package_bucket" {
  type = string
}

variable "deployment_package_key" {
  type = string
}

variable "cloudwatch_event_rule" {
  type    = string
  default = "weekly-report"
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
