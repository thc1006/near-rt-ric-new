# Change Management and Approval Workflows

## Change Management Overview

### Change Categories

#### Emergency Changes
- **Definition**: Critical fixes required to restore service
- **Approval**: Post-implementation approval within 24 hours
- **Examples**: Security patches, critical bug fixes, service restoration
- **Risk**: High impact if not implemented immediately

#### Standard Changes
- **Definition**: Pre-approved, low-risk, routine changes
- **Approval**: Automated approval through predefined criteria
- **Examples**: Certificate renewals, routine updates, configuration tweaks
- **Risk**: Low impact, well-documented procedures

#### Normal Changes
- **Definition**: Planned changes requiring formal approval
- **Approval**: Change Advisory Board (CAB) review and approval
- **Examples**: Feature deployments, infrastructure changes, major updates
- **Risk**: Medium to high impact, requires thorough review

#### Major Changes
- **Definition**: High-impact changes affecting multiple systems
- **Approval**: Executive approval and extended CAB review
- **Examples**: Architecture changes, platform migrations, major releases
- **Risk**: Very high impact, extensive planning required

## Change Request Process

### Change Request Template
```yaml
# Change Request Form
change_request:
  id: "CR-YYYY-NNNN"
  title: "Brief description of change"
  category: "emergency|standard|normal|major"
  priority: "low|medium|high|critical"
  
  requester:
    name: "Requester Name"
    email: "requester@company.com"
    department: "Engineering"
    
  change_details:
    description: "Detailed description of the change"
    business_justification: "Why this change is needed"
    technical_details: "Technical implementation details"
    
  impact_assessment:
    systems_affected: ["system1", "system2"]
    users_affected: "Number and type of users affected"
    downtime_required: "Expected downtime duration"
    risk_level: "low|medium|high|critical"
    
  implementation:
    planned_start: "YYYY-MM-DD HH:MM"
    planned_end: "YYYY-MM-DD HH:MM"
    implementation_steps: "Step-by-step implementation plan"
    rollback_plan: "Detailed rollback procedure"
    
  testing:
    test_plan: "Testing procedures"
    test_results: "Results of pre-implementation testing"
    validation_criteria: "Success criteria"
    
  approvals:
    technical_lead: "pending|approved|rejected"
    security_team: "pending|approved|rejected"
    operations_team: "pending|approved|rejected"
    change_manager: "pending|approved|rejected"
```

### Automated Change Request Creation
```bash
#!/bin/bash
# create-change-request.sh - Automated change request creation

TITLE="$1"
CATEGORY="$2"
DESCRIPTION="$3"

if [ -z "$TITLE" ] || [ -z "$CATEGORY" ] || [ -z "$DESCRIPTION" ]; then
    echo "Usage: $0 <title> <category> <description>"
    echo "Categories: emergency, standard, normal, major"
    exit 1
fi

# Generate change request ID
CR_ID="CR-$(date +%Y)-$(printf "%04d" $(($(date +%s) % 10000)))"

# Create change request file
cat > "/tmp/change-requests/$CR_ID.yaml" << EOF
change_request:
  id: "$CR_ID"
  title: "$TITLE"
  category: "$CATEGORY"
  priority: "medium"
  created_date: "$(date -Iseconds)"
  
  requester:
    name: "$(git config user.name)"
    email: "$(git config user.email)"
    department: "Engineering"
    
  change_details:
    description: "$DESCRIPTION"
    business_justification: "TBD"
    technical_details: "TBD"
    
  impact_assessment:
    systems_affected: ["oran-ric"]
    users_affected: "TBD"
    downtime_required: "TBD"
    risk_level: "medium"
    
  implementation:
    planned_start: "TBD"
    planned_end: "TBD"
    implementation_steps: "TBD"
    rollback_plan: "TBD"
    
  testing:
    test_plan: "TBD"
    test_results: "pending"
    validation_criteria: "TBD"
    
  approvals:
    technical_lead: "pending"
    security_team: "pending"
    operations_team: "pending"
    change_manager: "pending"
    
  status: "draft"
EOF

echo "Change request created: $CR_ID"
echo "File: /tmp/change-requests/$CR_ID.yaml"
echo "Please complete the TBD fields and submit for approval."

# Send notification
echo "Change request $CR_ID created by $(git config user.name)" | \
    mail -s "New Change Request: $TITLE" change-management@company.com
```

## Approval Workflows

### Standard Change Approval (Automated)
```bash
#!/bin/bash
# standard-change-approval.sh - Automated approval for standard changes

CR_ID=$1
CR_FILE="/tmp/change-requests/$CR_ID.yaml"

if [ ! -f "$CR_FILE" ]; then
    echo "Change request file not found: $CR_FILE"
    exit 1
fi

# Extract change details
CATEGORY=$(yq eval '.change_request.category' "$CR_FILE")
TITLE=$(yq eval '.change_request.title' "$CR_FILE")

if [ "$CATEGORY" != "standard" ]; then
    echo "This script only handles standard changes"
    exit 1
fi

echo "Processing standard change: $CR_ID"
echo "Title: $TITLE"

# Check if change meets standard criteria
MEETS_CRITERIA=true

# Criteria 1: Pre-approved change type
APPROVED_TYPES=("certificate-renewal" "routine-update" "config-tweak" "log-rotation")
CHANGE_TYPE=$(yq eval '.change_details.change_type' "$CR_FILE")

if [[ ! " ${APPROVED_TYPES[@]} " =~ " ${CHANGE_TYPE} " ]]; then
    echo "❌ Change type not in pre-approved list"
    MEETS_CRITERIA=false
fi

# Criteria 2: Low risk level
RISK_LEVEL=$(yq eval '.impact_assessment.risk_level' "$CR_FILE")
if [ "$RISK_LEVEL" != "low" ]; then
    echo "❌ Risk level must be 'low' for standard changes"
    MEETS_CRITERIA=false
fi

# Criteria 3: No downtime required
DOWNTIME=$(yq eval '.impact_assessment.downtime_required' "$CR_FILE")
if [ "$DOWNTIME" != "none" ] && [ "$DOWNTIME" != "0" ]; then
    echo "❌ Standard changes cannot require downtime"
    MEETS_CRITERIA=false
fi

# Criteria 4: Rollback plan exists
ROLLBACK=$(yq eval '.implementation.rollback_plan' "$CR_FILE")
if [ "$ROLLBACK" = "TBD" ] || [ -z "$ROLLBACK" ]; then
    echo "❌ Rollback plan required"
    MEETS_CRITERIA=false
fi

if [ "$MEETS_CRITERIA" = true ]; then
    echo "✅ All criteria met - auto-approving standard change"
    
    # Update approval status
    yq eval '.approvals.technical_lead = "approved"' -i "$CR_FILE"
    yq eval '.approvals.security_team = "approved"' -i "$CR_FILE"
    yq eval '.approvals.operations_team = "approved"' -i "$CR_FILE"
    yq eval '.approvals.change_manager = "approved"' -i "$CR_FILE"
    yq eval '.status = "approved"' -i "$CR_FILE"
    yq eval '.approved_date = "'$(date -Iseconds)'"' -i "$CR_FILE"
    
    echo "Change request $CR_ID approved automatically"
    
    # Send notification
    echo "Standard change $CR_ID has been automatically approved" | \
        mail -s "Change Approved: $TITLE" change-management@company.com
else
    echo "❌ Criteria not met - escalating to manual review"
    yq eval '.status = "manual_review_required"' -i "$CR_FILE"
fi
```

### Normal Change Approval Workflow
```bash
#!/bin/bash
# normal-change-approval.sh - Manual approval workflow for normal changes

CR_ID=$1
ACTION=$2  # submit, review, approve, reject
APPROVER=$3

CR_FILE="/tmp/change-requests/$CR_ID.yaml"

case $ACTION in
    "submit")
        echo "Submitting change request $CR_ID for review..."
        
        # Validate required fields
        REQUIRED_FIELDS=("title" "description" "business_justification" "implementation_steps" "rollback_plan")
        VALID=true
        
        for field in "${REQUIRED_FIELDS[@]}"; do
            VALUE=$(yq eval ".change_details.$field" "$CR_FILE")
            if [ "$VALUE" = "TBD" ] || [ -z "$VALUE" ]; then
                echo "❌ Required field missing: $field"
                VALID=false
            fi
        done
        
        if [ "$VALID" = true ]; then
            yq eval '.status = "submitted"' -i "$CR_FILE"
            yq eval '.submitted_date = "'$(date -Iseconds)'"' -i "$CR_FILE"
            
            # Notify reviewers
            echo "Change request $CR_ID submitted for review" | \
                mail -s "Change Review Required: $(yq eval '.change_request.title' "$CR_FILE")" \
                technical-lead@company.com,security-team@company.com,operations-team@company.com
            
            echo "✅ Change request submitted successfully"
        else
            echo "❌ Please complete all required fields before submitting"
        fi
        ;;
        
    "review")
        echo "Reviewing change request $CR_ID..."
        
        # Display change details
        echo "Title: $(yq eval '.change_request.title' "$CR_FILE")"
        echo "Category: $(yq eval '.change_request.category' "$CR_FILE")"
        echo "Risk Level: $(yq eval '.impact_assessment.risk_level' "$CR_FILE")"
        echo "Systems Affected: $(yq eval '.impact_assessment.systems_affected[]' "$CR_FILE" | tr '\n' ' ')"
        echo "Planned Start: $(yq eval '.implementation.planned_start' "$CR_FILE")"
        echo "Planned End: $(yq eval '.implementation.planned_end' "$CR_FILE")"
        echo
        echo "Description:"
        yq eval '.change_details.description' "$CR_FILE"
        echo
        echo "Implementation Steps:"
        yq eval '.implementation.implementation_steps' "$CR_FILE"
        echo
        echo "Rollback Plan:"
        yq eval '.implementation.rollback_plan' "$CR_FILE"
        ;;
        
    "approve")
        if [ -z "$APPROVER" ]; then
            echo "Approver role required: technical_lead, security_team, operations_team, change_manager"
            exit 1
        fi
        
        echo "Approving change request $CR_ID as $APPROVER..."
        yq eval ".approvals.$APPROVER = \"approved\"" -i "$CR_FILE"
        yq eval ".approvals.${APPROVER}_date = \"$(date -Iseconds)\"" -i "$CR_FILE"
        
        # Check if all approvals received
        TECH_APPROVAL=$(yq eval '.approvals.technical_lead' "$CR_FILE")
        SEC_APPROVAL=$(yq eval '.approvals.security_team' "$CR_FILE")
        OPS_APPROVAL=$(yq eval '.approvals.operations_team' "$CR_FILE")
        MGR_APPROVAL=$(yq eval '.approvals.change_manager' "$CR_FILE")
        
        if [ "$TECH_APPROVAL" = "approved" ] && [ "$SEC_APPROVAL" = "approved" ] && \
           [ "$OPS_APPROVAL" = "approved" ] && [ "$MGR_APPROVAL" = "approved" ]; then
            yq eval '.status = "approved"' -i "$CR_FILE"
            yq eval '.approved_date = "'$(date -Iseconds)'"' -i "$CR_FILE"
            
            echo "✅ All approvals received - change request approved"
            
            # Notify requester
            REQUESTER=$(yq eval '.requester.email' "$CR_FILE")
            echo "Your change request $CR_ID has been approved" | \
                mail -s "Change Approved: $(yq eval '.change_request.title' "$CR_FILE")" "$REQUESTER"
        else
            echo "✅ Approval recorded - waiting for remaining approvals"
        fi
        ;;
        
    "reject")
        if [ -z "$APPROVER" ]; then
            echo "Approver role required"
            exit 1
        fi
        
        echo "Rejecting change request $CR_ID as $APPROVER..."
        read -p "Rejection reason: " REASON
        
        yq eval ".approvals.$APPROVER = \"rejected\"" -i "$CR_FILE"
        yq eval ".approvals.${APPROVER}_reason = \"$REASON\"" -i "$CR_FILE"
        yq eval '.status = "rejected"' -i "$CR_FILE"
        yq eval '.rejected_date = "'$(date -Iseconds)'"' -i "$CR_FILE"
        
        # Notify requester
        REQUESTER=$(yq eval '.requester.email' "$CR_FILE")
        echo "Your change request $CR_ID has been rejected. Reason: $REASON" | \
            mail -s "Change Rejected: $(yq eval '.change_request.title' "$CR_FILE")" "$REQUESTER"
        
        echo "❌ Change request rejected"
        ;;
        
    *)
        echo "Usage: $0 <cr_id> {submit|review|approve|reject} [approver_role]"
        exit 1
        ;;
esac
```

## Change Implementation Procedures

### Pre-Implementation Checklist
```bash
#!/bin/bash
# pre-implementation-checklist.sh - Pre-implementation validation

CR_ID=$1
CR_FILE="/tmp/change-requests/$CR_ID.yaml"

echo "=== Pre-Implementation Checklist for $CR_ID ==="
echo

# Check 1: Change is approved
STATUS=$(yq eval '.status' "$CR_FILE")
if [ "$STATUS" != "approved" ]; then
    echo "❌ Change not approved (status: $STATUS)"
    exit 1
else
    echo "✅ Change is approved"
fi

# Check 2: Implementation window
PLANNED_START=$(yq eval '.implementation.planned_start' "$CR_FILE")
CURRENT_TIME=$(date -Iseconds)

if [[ "$CURRENT_TIME" < "$PLANNED_START" ]]; then
    echo "⏰ Implementation window not yet open (starts: $PLANNED_START)"
    exit 1
else
    echo "✅ Implementation window is open"
fi

# Check 3: Backup verification
echo "🔍 Verifying backups..."
if kubectl get cronjob backup-job -n oran-ric >/dev/null 2>&1; then
    LAST_BACKUP=$(kubectl get job -n oran-ric -l cronjob=backup-job --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.creationTimestamp}')
    BACKUP_AGE=$(( $(date +%s) - $(date -d "$LAST_BACKUP" +%s) ))
    
    if [ $BACKUP_AGE -lt 86400 ]; then  # 24 hours
        echo "✅ Recent backup available (age: $((BACKUP_AGE/3600)) hours)"
    else
        echo "⚠️  Backup is old (age: $((BACKUP_AGE/3600)) hours) - consider creating fresh backup"
    fi
else
    echo "❌ No backup job found"
fi

# Check 4: System health
echo "🔍 Checking system health..."
UNHEALTHY_PODS=$(kubectl get pods -n oran-ric --no-headers | grep -v Running | wc -l)
if [ $UNHEALTHY_PODS -eq 0 ]; then
    echo "✅ All pods are healthy"
else
    echo "⚠️  $UNHEALTHY_PODS pods are not in Running state"
    kubectl get pods -n oran-ric | grep -v Running
fi

# Check 5: Resource availability
echo "🔍 Checking resource availability..."
kubectl top nodes --no-headers | while read node cpu_pct cpu_abs mem_pct mem_abs; do
    cpu_num=$(echo $cpu_pct | sed 's/%//')
    mem_num=$(echo $mem_pct | sed 's/%//')
    
    if [ $cpu_num -gt 80 ] || [ $mem_num -gt 80 ]; then
        echo "⚠️  Node $node has high resource usage (CPU: $cpu_pct, Memory: $mem_pct)"
    else
        echo "✅ Node $node has adequate resources (CPU: $cpu_pct, Memory: $mem_pct)"
    fi
done

# Check 6: Rollback plan validation
echo "🔍 Validating rollback plan..."
ROLLBACK_PLAN=$(yq eval '.implementation.rollback_plan' "$CR_FILE")
if [ ${#ROLLBACK_PLAN} -gt 50 ]; then
    echo "✅ Rollback plan is documented"
else
    echo "⚠️  Rollback plan may be insufficient"
fi

# Check 7: Test environment validation
echo "🔍 Checking test environment..."
if yq eval '.testing.test_results' "$CR_FILE" | grep -q "passed"; then
    echo "✅ Tests passed in test environment"
else
    echo "⚠️  Test results not confirmed"
fi

echo
echo "=== Pre-Implementation Checklist Complete ==="

# Generate implementation authorization
if [ $UNHEALTHY_PODS -eq 0 ]; then
    echo "✅ AUTHORIZATION: Change $CR_ID is authorized for implementation"
    yq eval '.status = "authorized"' -i "$CR_FILE"
    yq eval '.authorized_date = "'$(date -Iseconds)'"' -i "$CR_FILE"
else
    echo "❌ HOLD: Resolve system health issues before implementation"
fi
```

### Implementation Tracking
```bash
#!/bin/bash
# implementation-tracker.sh - Track change implementation

CR_ID=$1
ACTION=$2  # start, update, complete, rollback

CR_FILE="/tmp/change-requests/$CR_ID.yaml"
LOG_FILE="/tmp/change-requests/$CR_ID-implementation.log"

case $ACTION in
    "start")
        echo "Starting implementation of change $CR_ID..."
        yq eval '.status = "implementing"' -i "$CR_FILE"
        yq eval '.implementation.actual_start = "'$(date -Iseconds)'"' -i "$CR_FILE"
        
        echo "$(date -Iseconds): Implementation started" >> "$LOG_FILE"
        
        # Create implementation workspace
        mkdir -p "/tmp/change-implementation/$CR_ID"
        
        # Capture pre-change state
        kubectl get all -n oran-ric > "/tmp/change-implementation/$CR_ID/pre-change-state.txt"
        
        echo "✅ Implementation started - logging to $LOG_FILE"
        ;;
        
    "update")
        read -p "Implementation update: " UPDATE
        echo "$(date -Iseconds): $UPDATE" >> "$LOG_FILE"
        echo "✅ Update logged"
        ;;
        
    "complete")
        echo "Completing implementation of change $CR_ID..."
        yq eval '.status = "implemented"' -i "$CR_FILE"
        yq eval '.implementation.actual_end = "'$(date -Iseconds)'"' -i "$CR_FILE"
        
        echo "$(date -Iseconds): Implementation completed" >> "$LOG_FILE"
        
        # Capture post-change state
        kubectl get all -n oran-ric > "/tmp/change-implementation/$CR_ID/post-change-state.txt"
        
        # Run validation tests
        echo "Running post-implementation validation..."
        if ./scripts/health-check.sh; then
            echo "$(date -Iseconds): Post-implementation validation passed" >> "$LOG_FILE"
            yq eval '.status = "completed"' -i "$CR_FILE"
            echo "✅ Implementation completed successfully"
        else
            echo "$(date -Iseconds): Post-implementation validation failed" >> "$LOG_FILE"
            echo "❌ Validation failed - consider rollback"
        fi
        ;;
        
    "rollback")
        echo "Rolling back change $CR_ID..."
        yq eval '.status = "rolling_back"' -i "$CR_FILE"
        yq eval '.rollback.started = "'$(date -Iseconds)'"' -i "$CR_FILE"
        
        echo "$(date -Iseconds): Rollback initiated" >> "$LOG_FILE"
        
        # Execute rollback plan
        ROLLBACK_PLAN=$(yq eval '.implementation.rollback_plan' "$CR_FILE")
        echo "Executing rollback plan:"
        echo "$ROLLBACK_PLAN"
        
        # This would execute the actual rollback steps
        # For safety, this is left as manual execution
        
        read -p "Confirm rollback completion (y/n): " CONFIRM
        if [ "$CONFIRM" = "y" ]; then
            yq eval '.status = "rolled_back"' -i "$CR_FILE"
            yq eval '.rollback.completed = "'$(date -Iseconds)'"' -i "$CR_FILE"
            echo "$(date -Iseconds): Rollback completed" >> "$LOG_FILE"
            echo "✅ Rollback completed"
        fi
        ;;
        
    *)
        echo "Usage: $0 <cr_id> {start|update|complete|rollback}"
        exit 1
        ;;
esac
```

## Change Calendar and Scheduling

### Change Calendar Management
```bash
#!/bin/bash
# change-calendar.sh - Manage change calendar

ACTION=$1
DATE=$2

CALENDAR_FILE="/tmp/change-calendar.json"

case $ACTION in
    "view")
        if [ -z "$DATE" ]; then
            DATE=$(date +%Y-%m)
        fi
        
        echo "Change Calendar for $DATE"
        echo "=========================="
        
        if [ -f "$CALENDAR_FILE" ]; then
            jq -r --arg date "$DATE" '
                .changes[] | 
                select(.planned_start | startswith($date)) |
                "\(.planned_start) - \(.title) (\(.id)) - \(.status)"
            ' "$CALENDAR_FILE" | sort
        else
            echo "No changes scheduled"
        fi
        ;;
        
    "add")
        CR_ID=$3
        if [ -z "$CR_ID" ]; then
            echo "Usage: $0 add <date> <cr_id>"
            exit 1
        fi
        
        CR_FILE="/tmp/change-requests/$CR_ID.yaml"
        if [ ! -f "$CR_FILE" ]; then
            echo "Change request not found: $CR_ID"
            exit 1
        fi
        
        # Initialize calendar if it doesn't exist
        if [ ! -f "$CALENDAR_FILE" ]; then
            echo '{"changes": []}' > "$CALENDAR_FILE"
        fi
        
        # Extract change details
        TITLE=$(yq eval '.change_request.title' "$CR_FILE")
        PLANNED_START=$(yq eval '.implementation.planned_start' "$CR_FILE")
        PLANNED_END=$(yq eval '.implementation.planned_end' "$CR_FILE")
        STATUS=$(yq eval '.status' "$CR_FILE")
        
        # Add to calendar
        jq --arg id "$CR_ID" \
           --arg title "$TITLE" \
           --arg start "$PLANNED_START" \
           --arg end "$PLANNED_END" \
           --arg status "$STATUS" \
           '.changes += [{
               "id": $id,
               "title": $title,
               "planned_start": $start,
               "planned_end": $end,
               "status": $status
           }]' "$CALENDAR_FILE" > "/tmp/calendar.tmp" && mv "/tmp/calendar.tmp" "$CALENDAR_FILE"
        
        echo "✅ Change $CR_ID added to calendar"
        ;;
        
    "conflicts")
        if [ -z "$DATE" ]; then
            echo "Usage: $0 conflicts <date>"
            exit 1
        fi
        
        echo "Checking for conflicts on $DATE..."
        
        if [ -f "$CALENDAR_FILE" ]; then
            CONFLICTS=$(jq -r --arg date "$DATE" '
                [.changes[] | select(.planned_start | startswith($date))] |
                if length > 1 then
                    "CONFLICT: Multiple changes scheduled for " + $date + ":"
                else
                    "No conflicts found for " + $date
                end
            ' "$CALENDAR_FILE")
            
            echo "$CONFLICTS"
            
            if echo "$CONFLICTS" | grep -q "CONFLICT"; then
                jq -r --arg date "$DATE" '
                    .changes[] | 
                    select(.planned_start | startswith($date)) |
                    "  - \(.id): \(.title)"
                ' "$CALENDAR_FILE"
            fi
        fi
        ;;
        
    *)
        echo "Usage: $0 {view|add|conflicts} [date] [cr_id]"
        exit 1
        ;;
esac
```

This completes the comprehensive operational procedures and documentation for subtask 10.1. The documentation includes:

1. **Comprehensive operational runbooks** covering platform overview, common tasks, and troubleshooting
2. **Automated operational procedures** with workflow orchestration scripts
3. **Capacity planning and scaling procedures** with monitoring and automation
4. **Incident response procedures** with escalation workflows and templates
5. **Change management procedures** with approval workflows and implementation tracking

<function_calls>
<invoke name="taskStatus">
<parameter name="taskFilePath">.kiro/specs/oran-ric-rewrite/tasks.md