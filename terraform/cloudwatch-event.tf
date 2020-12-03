resource "aws_cloudwatch_event_rule" "event_rule" {
  name                = var.cloudwatch_event_rule_name
  schedule_expression = var.cloudwatch_event_schedule_expression
}

resource "aws_lambda_permission" "allow_cloudwatch" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.lambda.arn
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.event_rule.arn
}
