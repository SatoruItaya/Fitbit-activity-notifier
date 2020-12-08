resource "aws_cloudwatch_event_rule" "event_rule" {
  name                = var.cloudwatch_event_rule_name
  schedule_expression = var.cloudwatch_event_schedule_expression
}

resource "aws_cloudwatch_event_target" "event_target" {
  target_id = var.lambda_function_name
  rule      = aws_cloudwatch_event_rule.event_rule.name
  arn       = aws_lambda_function.lambda.arn
  input     = "{\"start_date\":\"${var.fitbit_start_date_of_use}\"}"
}

resource "aws_lambda_permission" "allow_cloudwatch" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.lambda.arn
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.event_rule.arn
}
