# Read docker-compose.yml
with open('docker-compose.yml', 'r', encoding='utf-8') as f:
    lines = f.readlines()

# Find and replace calendar-service ports
in_calendar_service = False
for i, line in enumerate(lines):
    if 'calendar-service:' in line:
        in_calendar_service = True
    elif in_calendar_service and line.strip().startswith('- "50059:50059"'):
        lines[i] = line.replace('50059:50059', '50058:50058')
    elif in_calendar_service and 'GRPC_PORT: 50059' in line:
        lines[i] = line.replace('GRPC_PORT: 50059', 'GRPC_PORT: 50058')
        in_calendar_service = False  # Reset after finding GRPC_PORT
    elif in_calendar_service and line.strip() and not line.strip().startswith('-') and 'GRPC_PORT' not in line and 'ports:' not in line and 'container_name' not in line:
        # We've moved past the calendar-service section
        in_calendar_service = False

# Write back
with open('docker-compose.yml', 'w', encoding='utf-8') as f:
    f.writelines(lines)

print("Fixed calendar-service port to 50058")
