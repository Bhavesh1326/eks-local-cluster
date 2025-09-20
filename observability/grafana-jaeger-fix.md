# Jaeger Data Source Configuration Fix

## Issue
Jaeger returns HTML instead of JSON because it's configured with base path `/jaeger/`

## Solution
In Grafana, when adding Jaeger data source:

1. **URL**: `http://jaeger-query.observability.svc.cluster.local:16686`
2. **Advanced settings**:
   - **Derived fields**: Leave empty for now
   - **Trace to logs**: Can configure later
   - **Node graph**: Can configure later

## Alternative: Direct API Access
Try using the API path directly:
- **URL**: `http://jaeger-query.observability.svc.cluster.local:16686/api`

## Manual Testing
To test if Jaeger is working, access: http://localhost:16686/jaeger

The error you see in Grafana ("invalid character '<'") happens because:
1. Grafana expects JSON from `/api/services`
2. But Jaeger is returning HTML due to the base path configuration
3. This is a common issue with Jaeger all-in-one deployments
