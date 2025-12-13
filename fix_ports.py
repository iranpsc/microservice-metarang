import re

# Read docker-compose.yml
with open('docker-compose.yml', 'r', encoding='utf-8') as f:
    content = f.read()

# Fix calendar-service: change 50059 to 50058
# Find the calendar-service section and replace ports and GRPC_PORT
content = re.sub(
    r'(calendar-service:.*?ports:\s+- ")(\d+):(\d+)(".*?GRPC_PORT: )(\d+)',
    r'\g<1>50058:50058\g<4>50058',
    content,
    flags=re.DOTALL
)

# Fix storage-service: change 50060 to 50059
# Find the storage-service section and replace ports and GRPC_PORT
content = re.sub(
    r'(storage-service:.*?ports:\s+- ")(\d+):(\d+)(".*?GRPC_PORT: )(\d+)',
    r'\g<1>50059:50059\g<4>50059',
    content,
    flags=re.DOTALL
)

# Write back
with open('docker-compose.yml', 'w', encoding='utf-8') as f:
    f.write(content)

print("Fixed port configurations")
