# Cron-Scheduled Orders

Artifact Conduit supports scheduling Orders to run automatically on a recurring schedule using cron expressions. This feature enables you to process artifacts on a regular basis without manual intervention.

## Overview

Instead of creating an Order once and running it immediately, you can define a cron schedule that automatically creates and executes ArtifactWorkflows at specified times. This is useful for:

- **Periodic scanning**: Run security scans on artifacts at regular intervals
- **Scheduled backups**: Process and store artifacts on a schedule
- **Recurring transformations**: Apply transformations to artifacts at set times
- **Compliance workflows**: Execute compliance-related artifact processing on a schedule

## How it Works

When you define a cron schedule in an Order, Artifact Conduit:

1. Creates a CronWorkflow that represents the schedule
2. The CronWorkflow automatically creates Workflows at the scheduled times
3. Each Workflow processes your artifacts according to the Order specification
4. The status of each run is tracked in the Order (pay attention the `failed` and `succeeded` counts)

## Configuration

Cron scheduling in Orders can be configured at two levels:

### 1. Default Schedule for All Artifacts

Define a schedule that applies to all artifacts in the Order using `spec.defaults.cron`:

```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: Order
metadata:
  name: scheduled-scan
  namespace: default
spec:
  defaults:
    srcRef:
      name: docker-registry
    dstRef:
      name: internal-registry
    cron:
      schedules:
        - "0 2 * * *"  # Run daily at 2 AM
      timezone: "UTC"
  artifacts:
    - type: oci
      spec:
        image: "my-app:latest"
```

### 2. Per-Artifact Schedule

Override the default schedule for specific artifacts using `spec.artifacts[].cron`:

```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: Order
metadata:
  name: mixed-schedules
  namespace: default
spec:
  defaults:
    srcRef:
      name: docker-registry
    dstRef:
      name: internal-registry
    cron:
      schedules:
        - "0 2 * * *"  # Default: daily at 2 AM
  artifacts:
    - type: oci
      spec:
        image: "app-a:latest"
      # Uses default schedule from spec.defaults.cron
    
    - type: oci
      spec:
        image: "app-b:latest"
      cron:
        schedules:
          - "0 3 * * 0"  # Override: weekly on Sunday at 3 AM
        timezone: "Europe/Berlin"
```

## Cron Schedule Format

The cron schedule supports standard cron expressions with the following formats:

### Standard Cron Format
```
minute hour day month weekday
```

Examples:
- `"0 2 * * *"` - Daily at 2:00 AM
- `"0 */4 * * *"` - Every 4 hours
- `"0 9 * * 1-5"` - Weekdays at 9:00 AM
- `"0 0 1 * *"` - First day of each month at midnight
- `"0 12 * * 0"` - Every Sunday at noon

### Predefined Expressions
- `"@yearly"` or `"@annually"` - January 1 at 00:00
- `"@monthly"` - First day of month at 00:00
- `"@weekly"` - Sunday at 00:00
- `"@daily"` or `"@midnight"` - Daily at 00:00
- `"@hourly"` - Every hour at :00

### Duration Format
- `"@every 1h30m"` - Every 1 hour 30 minutes
- `"@every 5m"` - Every 5 minutes

## Configuration Options

### Timezone

Specify the timezone for schedule calculations:

```yaml
cron:
  schedules:
    - "0 2 * * *"
  timezone: "America/New_York"  # Use America/New_York timezone
```

Supported timezone formats follow IANA timezone identifiers (e.g., `UTC`, `Europe/Berlin`, `Asia/Tokyo`).

**Default**: System's local timezone if not specified.

### Starting Deadline Seconds

Limit how long a scheduled run can be delayed if it misses its scheduled time:

```yaml
cron:
  schedules:
    - "0 2 * * *"
  startingDeadlineSeconds: 3600  # Allow up to 1 hour delay
```

This is useful if your cluster experiences temporary unavailability. If a run is missed, it will only execute if the deadline hasn't passed.

**Default**: No deadline (run will execute whenever the cluster is available).

### Conditional Execution

Execute the schedule only when a specific condition is met:

```yaml
cron:
  schedules:
    - "0 2 * * *"
  when: "some.condition == 'true'"  # Pseudo-code
```

### Multiple Schedules

Run the same artifacts on multiple schedules:

```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: Order
metadata:
  name: multi-schedule-scan
  namespace: default
spec:
  defaults:
    srcRef:
      name: registry
    dstRef:
      name: storage
  artifacts:
    - type: sbom
      cron:
        schedules:
          - "0 2 * * 1"     # Monday at 2 AM
          - "0 2 * * 4"     # Thursday at 2 AM
        timezone: "UTC"
```

### Time-Sensitive Scanning

Scan frequently during business hours, less often outside:

```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: Order
metadata:
  name: business-hours-scan
  namespace: default
spec:
  defaults:
    srcRef:
      name: registry
    dstRef:
      name: reports
  artifacts:
    - type: compliance-check
      cron:
        schedules:
          - "0 */2 * * 1-5"  # Every 2 hours on weekdays
```

## Best Practices

1. **Use UTC Timezone**: For consistency across teams, use UTC timezone unless you have a specific reason not to.

2. **Avoid Peak Hours**: Schedule resource-intensive operations outside peak business hours to avoid cluster congestion.

3. **Set Appropriate Deadlines**: Use `startingDeadlineSeconds` for important schedules to ensure they eventually run.

4. **Monitor Long Runs**: For Orders that might take a long time, be aware that the next scheduled run might start before the previous one completes.

5. **Test Schedules**: Before deploying to production, test your cron expressions to ensure they match your expectations.

6. **Document Your Schedules**: Use annotations to document why a particular schedule was chosen:
   ```yaml
   metadata:
     annotations:
       schedule-reason: "Daily compliance check at 2 AM UTC"
   ```

