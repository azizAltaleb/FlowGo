#!/bin/bash

echo "Deploying BPMN workflow..."
echo "Request Body (order.bpmn):"
cat order.bpmn
echo "----------------------------------------"

# curl sending XML file
DEPLOY_RESPONSE=$(curl -s -X POST http://localhost:8080/workflows \
  -H "Content-Type: application/xml" \
  --data-binary @order.bpmn)
echo "Response: $DEPLOY_RESPONSE"
WORKFLOW_ID=$(echo $DEPLOY_RESPONSE | jq -r '.id')
echo "Workflow ID: $WORKFLOW_ID"
echo "----------------------------------------"

if [ "$WORKFLOW_ID" == "null" ]; then
    echo "Failed to deploy workflow"
    exit 1
fi

echo "Starting instance..."
START_RESPONSE=$(curl -s -X POST http://localhost:8080/instances \
  -H "Content-Type: application/json" \
  -d "{\"workflow_id\": \"$WORKFLOW_ID\", \"context\": {\"customer\": \"Bob\", \"amount\": 250}}")
echo "Response: $START_RESPONSE"
INSTANCE_ID=$(echo $START_RESPONSE | jq -r '.id')
echo "Instance ID: $INSTANCE_ID"
echo "----------------------------------------"

# The start event (StartEvent_1) is the first step.
# In our engine, it is a step of type START.
# We decided NOT to auto-complete it in the code yet (commented out).
# So we must complete it manually or check if it is auto-skipped.
# Wait, let's check the status first.

echo "Current Status (Should be at StartEvent_1):"
curl -s http://localhost:8080/instances/$INSTANCE_ID | jq .
echo "----------------------------------------"

echo "Completing 'StartEvent_1'..."
curl -s -X POST http://localhost:8080/instances/$INSTANCE_ID/complete
echo ""

echo "Current Status (Should be at validate_order):"
curl -s http://localhost:8080/instances/$INSTANCE_ID | jq .
echo "----------------------------------------"

echo "Completing 'validate_order'..."
curl -s -X POST http://localhost:8080/instances/$INSTANCE_ID/complete
echo ""

echo "Current Status (Should be at process_payment):"
curl -s http://localhost:8080/instances/$INSTANCE_ID | jq .
echo "----------------------------------------"

echo "Completing 'process_payment'..."
curl -s -X POST http://localhost:8080/instances/$INSTANCE_ID/complete
echo ""

echo "Current Status (Completed - EndEvent_1):"
curl -s http://localhost:8080/instances/$INSTANCE_ID | jq .
echo "----------------------------------------"
