resource "aws_cloudwatch_event_rule" "event_rule" {
  name                = var.cloudwatch_event_rule
  schedule_expression = "cron(0 3 ? * SUN *)"
}

resource "aws_lambda_permission" "allow_cloudwatch" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.lambda.arn
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.event_rule.arn
}
