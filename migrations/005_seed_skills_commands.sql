-- Migration 005: Seed initial skills and commands
-- Run with: psql -d mcp_serve -f migrations/005_seed_skills_commands.sql

-- ============ Seed Skills ============

INSERT INTO skills (name, version, description, category, content, tags, status, is_system, reputation_score) VALUES

-- kubectl skill
('kubectl', '1.0.0', 'Kubernetes command-line tool for cluster management, pod operations, and debugging',
'devops',
'# kubectl - Kubernetes CLI Reference

## Common Commands

### Cluster Info
```bash
kubectl cluster-info                    # Display cluster info
kubectl get nodes                       # List all nodes
kubectl describe node <node-name>       # Show node details
```

### Pod Operations
```bash
kubectl get pods                        # List pods in current namespace
kubectl get pods -A                     # List all pods in all namespaces
kubectl get pods -o wide               # Show pod IPs and nodes
kubectl describe pod <pod-name>        # Show pod details
kubectl logs <pod-name>                # View pod logs
kubectl logs -f <pod-name>             # Follow pod logs
kubectl logs <pod-name> -c <container> # Logs from specific container
kubectl exec -it <pod-name> -- /bin/sh # Shell into pod
```

### Deployments
```bash
kubectl get deployments                 # List deployments
kubectl describe deployment <name>     # Deployment details
kubectl scale deployment <name> --replicas=3  # Scale deployment
kubectl rollout status deployment/<name>      # Check rollout status
kubectl rollout restart deployment/<name>     # Restart deployment
kubectl rollout undo deployment/<name>        # Rollback deployment
```

### Services & Networking
```bash
kubectl get services                   # List services
kubectl get ingress                    # List ingress resources
kubectl port-forward svc/<name> 8080:80  # Port forward to service
kubectl port-forward pod/<name> 8080:80  # Port forward to pod
```

### Debugging
```bash
kubectl get events --sort-by=.metadata.creationTimestamp  # Recent events
kubectl top pods                       # Pod resource usage
kubectl top nodes                      # Node resource usage
kubectl debug pod/<name> -it --image=busybox  # Debug pod
```

### Config & Context
```bash
kubectl config get-contexts            # List contexts
kubectl config use-context <name>      # Switch context
kubectl config current-context         # Show current context
```

### Apply & Delete
```bash
kubectl apply -f <file.yaml>           # Apply configuration
kubectl delete -f <file.yaml>          # Delete resources
kubectl delete pod <name>              # Delete specific pod
kubectl delete pod <name> --grace-period=0 --force  # Force delete
```',
ARRAY['kubernetes', 'k8s', 'container', 'orchestration', 'devops', 'cli'],
'active', TRUE, 80.0),

-- docker skill
('docker-cli', '1.0.0', 'Docker command-line tool for container management, images, and networking',
'devops',
'# Docker CLI Reference

## Container Operations
```bash
docker ps                              # List running containers
docker ps -a                           # List all containers
docker run -d --name <name> <image>    # Run container detached
docker run -it <image> /bin/sh         # Run interactive shell
docker run -p 8080:80 <image>          # Run with port mapping
docker run -v /host:/container <image> # Run with volume mount
docker exec -it <container> /bin/sh    # Shell into container
docker logs <container>                # View container logs
docker logs -f <container>             # Follow logs
docker stop <container>                # Stop container
docker start <container>               # Start container
docker restart <container>             # Restart container
docker rm <container>                  # Remove container
docker rm -f <container>               # Force remove
```

## Image Operations
```bash
docker images                          # List images
docker pull <image>                    # Pull image
docker build -t <tag> .                # Build image
docker build -t <tag> -f Dockerfile.custom .  # Build with specific file
docker push <image>                    # Push to registry
docker tag <image> <new-tag>          # Tag image
docker rmi <image>                    # Remove image
docker image prune                    # Remove unused images
```

## Docker Compose
```bash
docker compose up                      # Start services
docker compose up -d                   # Start detached
docker compose up --build              # Rebuild and start
docker compose down                    # Stop and remove
docker compose logs -f                 # Follow all logs
docker compose ps                      # List services
docker compose exec <service> sh       # Shell into service
```

## Networking
```bash
docker network ls                      # List networks
docker network create <name>           # Create network
docker network inspect <name>          # Network details
docker network connect <net> <container>  # Connect container
```

## Volumes
```bash
docker volume ls                       # List volumes
docker volume create <name>            # Create volume
docker volume inspect <name>           # Volume details
docker volume prune                    # Remove unused volumes
```

## Cleanup
```bash
docker system prune                    # Remove all unused data
docker system prune -a                 # Remove all unused + images
docker system df                       # Show disk usage
```',
ARRAY['docker', 'container', 'devops', 'cli', 'containerization'],
'active', TRUE, 80.0),

-- curl skill
('curl', '1.0.0', 'Command-line tool for transferring data with URLs, HTTP requests, and API testing',
'cli',
'# curl - HTTP Client Reference

## Basic Requests
```bash
curl https://api.example.com           # GET request
curl -X POST https://api.example.com   # POST request
curl -X PUT https://api.example.com    # PUT request
curl -X DELETE https://api.example.com # DELETE request
curl -X PATCH https://api.example.com  # PATCH request
```

## Headers
```bash
curl -H "Content-Type: application/json" <url>
curl -H "Authorization: Bearer <token>" <url>
curl -H "X-Custom-Header: value" <url>
```

## Data/Body
```bash
# JSON body
curl -X POST -H "Content-Type: application/json" \
  -d ''{"key": "value"}'' <url>

# Form data
curl -X POST -d "name=value&other=data" <url>

# File upload
curl -X POST -F "file=@/path/to/file" <url>

# From file
curl -X POST -d @data.json <url>
```

## Authentication
```bash
curl -u username:password <url>        # Basic auth
curl -H "Authorization: Bearer <token>" <url>  # Bearer token
curl --oauth2-bearer <token> <url>     # OAuth2
```

## Output Options
```bash
curl -o output.json <url>              # Save to file
curl -O <url>                          # Save with original name
curl -s <url>                          # Silent mode
curl -v <url>                          # Verbose output
curl -i <url>                          # Include headers in output
curl -w "%{http_code}" <url>          # Show status code
```

## Common Patterns
```bash
# GET JSON and parse with jq
curl -s <url> | jq .

# POST JSON and check status
curl -s -w "%{http_code}" -o /dev/null -X POST \
  -H "Content-Type: application/json" \
  -d ''{"data": "value"}'' <url>

# Download with progress
curl -# -O <url>

# Follow redirects
curl -L <url>

# With timeout
curl --connect-timeout 5 --max-time 10 <url>

# Retry on failure
curl --retry 3 --retry-delay 2 <url>
```',
ARRAY['curl', 'http', 'api', 'cli', 'rest', 'testing'],
'active', TRUE, 75.0),

-- git skill
('git', '1.0.0', 'Version control system for tracking changes, branching, and collaboration',
'devops',
'# Git Reference

## Basic Commands
```bash
git init                               # Initialize repository
git clone <url>                        # Clone repository
git status                             # Check status
git add .                              # Stage all changes
git add <file>                         # Stage specific file
git commit -m "message"                # Commit with message
git push                               # Push to remote
git pull                               # Pull from remote
git fetch                              # Fetch without merge
```

## Branching
```bash
git branch                             # List branches
git branch <name>                      # Create branch
git checkout <branch>                  # Switch branch
git checkout -b <name>                 # Create and switch
git merge <branch>                     # Merge branch
git branch -d <name>                   # Delete branch
git branch -D <name>                   # Force delete
```

## History & Diff
```bash
git log                                # View history
git log --oneline                      # Compact history
git log --graph                        # Graph view
git diff                               # Unstaged changes
git diff --staged                      # Staged changes
git diff <branch1>..<branch2>          # Compare branches
git show <commit>                      # Show commit details
```

## Undoing Changes
```bash
git restore <file>                     # Discard changes
git restore --staged <file>            # Unstage file
git reset HEAD~1                       # Undo last commit (keep changes)
git reset --hard HEAD~1                # Undo last commit (discard)
git revert <commit>                    # Create reverting commit
git stash                              # Stash changes
git stash pop                          # Apply and remove stash
```

## Remote Operations
```bash
git remote -v                          # List remotes
git remote add origin <url>            # Add remote
git push -u origin <branch>            # Push and set upstream
git push --force-with-lease           # Safe force push
```

## Rebase
```bash
git rebase <branch>                    # Rebase onto branch
git rebase -i HEAD~3                   # Interactive rebase
git rebase --continue                  # Continue after conflict
git rebase --abort                     # Abort rebase
```',
ARRAY['git', 'version-control', 'devops', 'cli', 'collaboration'],
'active', TRUE, 80.0),

-- jq skill
('jq', '1.0.0', 'Command-line JSON processor for parsing, filtering, and transforming JSON data',
'cli',
'# jq - JSON Processor Reference

## Basic Usage
```bash
echo ''{"name": "test"}'' | jq .        # Pretty print
echo ''{"name": "test"}'' | jq .name    # Get field
cat file.json | jq .                   # Parse file
curl -s <url> | jq .                   # Parse API response
```

## Selection
```bash
jq .key                                # Get object key
jq .[0]                                # Get array element
jq .[]                                 # Iterate array
jq .key1.key2                          # Nested access
jq .key?                               # Optional (no error if missing)
```

## Filters
```bash
jq ''select(.age > 30)''                # Filter by condition
jq ''select(.name == "test")''          # Filter by value
jq ''select(.tags | contains(["a"]))''  # Filter by array contains
```

## Transformation
```bash
jq ''{name, age}''                       # Select specific fields
jq ''{newName: .name}''                  # Rename fields
jq ''. + {newField: "value"}''           # Add field
jq ''del(.field)''                       # Remove field
jq ''.[] | {name, id}''                  # Transform array elements
```

## Array Operations
```bash
jq ''length''                            # Array length
jq ''first''                             # First element
jq ''last''                              # Last element
jq ''reverse''                           # Reverse array
jq ''sort''                              # Sort array
jq ''sort_by(.field)''                   # Sort by field
jq ''unique''                            # Remove duplicates
jq ''group_by(.field)''                  # Group by field
jq ''map(.field)''                       # Map to field values
jq ''[.[] | select(.x)]''                # Filter and collect
```

## Output Formats
```bash
jq -r .name                            # Raw output (no quotes)
jq -c .                                # Compact output
jq -S .                                # Sort keys
jq --tab .                             # Tab indentation
```

## Common Patterns
```bash
# Extract array of names
jq -r ''.[].name''

# Create CSV-like output
jq -r ''.[] | [.name, .id] | @csv''

# Count items
jq ''[.[] | select(.active)] | length''

# Merge objects
jq -s ''add'' file1.json file2.json
```',
ARRAY['jq', 'json', 'cli', 'parsing', 'data'],
'active', TRUE, 70.0),

-- aws-cli skill
('aws-cli', '1.0.0', 'AWS Command Line Interface for managing AWS services',
'cloud',
'# AWS CLI Reference

## Configuration
```bash
aws configure                          # Interactive setup
aws configure list                     # Show config
aws sts get-caller-identity           # Check current identity
```

## S3
```bash
aws s3 ls                              # List buckets
aws s3 ls s3://bucket/                 # List bucket contents
aws s3 cp file.txt s3://bucket/       # Upload file
aws s3 cp s3://bucket/file.txt .      # Download file
aws s3 sync ./dir s3://bucket/dir     # Sync directory
aws s3 rm s3://bucket/file.txt        # Delete file
aws s3 rm s3://bucket/ --recursive    # Delete all
aws s3 mb s3://new-bucket             # Create bucket
aws s3 rb s3://bucket                 # Delete bucket
```

## EC2
```bash
aws ec2 describe-instances            # List instances
aws ec2 start-instances --instance-ids i-xxx
aws ec2 stop-instances --instance-ids i-xxx
aws ec2 terminate-instances --instance-ids i-xxx
aws ec2 describe-security-groups
aws ec2 describe-vpcs
```

## Lambda
```bash
aws lambda list-functions
aws lambda invoke --function-name <name> output.json
aws lambda update-function-code --function-name <name> --zip-file fileb://code.zip
aws lambda get-function --function-name <name>
```

## ECS
```bash
aws ecs list-clusters
aws ecs list-services --cluster <name>
aws ecs describe-services --cluster <name> --services <svc>
aws ecs update-service --cluster <name> --service <svc> --force-new-deployment
```

## CloudWatch
```bash
aws logs describe-log-groups
aws logs get-log-events --log-group-name <group> --log-stream-name <stream>
aws logs tail <log-group> --follow
```

## IAM
```bash
aws iam list-users
aws iam list-roles
aws iam get-user
aws iam list-attached-user-policies --user-name <name>
```',
ARRAY['aws', 'cloud', 'cli', 'devops', 'infrastructure'],
'active', TRUE, 75.0)

ON CONFLICT (name) DO NOTHING;

-- ============ Seed Commands ============

INSERT INTO commands (name, version, description, prompt, category, tags, status, is_system, reputation_score) VALUES

('review-pr', '1.0.0', 'Review a pull request for code quality, security, and best practices',
'Review the current pull request or code changes. Focus on:

1. **Code Quality**
   - Clean, readable code
   - Proper naming conventions
   - DRY principles
   - Error handling

2. **Security**
   - Input validation
   - Authentication/authorization
   - No hardcoded secrets
   - SQL injection prevention

3. **Performance**
   - Efficient algorithms
   - Database query optimization
   - Memory management

4. **Testing**
   - Test coverage
   - Edge cases handled

Provide specific, actionable feedback with code examples where helpful.',
'code', ARRAY['review', 'pr', 'quality'], 'active', TRUE, 75.0),

('fix-tests', '1.0.0', 'Fix failing tests in the project',
'Analyze and fix failing tests in the project:

1. First run the test suite to identify failures
2. For each failing test:
   - Understand what the test is checking
   - Identify why it''s failing
   - Determine if it''s a test bug or code bug
   - Fix appropriately

3. Run tests again to verify fixes
4. Ensure no regressions were introduced

Prioritize fixing tests that:
- Block CI/CD
- Test critical functionality
- Have clear failure messages',
'test', ARRAY['test', 'fix', 'debugging'], 'active', TRUE, 70.0),

('add-tests', '1.0.0', 'Add comprehensive tests for specified code',
'Add tests for the specified code or module:

1. **Unit Tests**
   - Test individual functions
   - Cover happy path and edge cases
   - Mock external dependencies

2. **Integration Tests** (if applicable)
   - Test component interactions
   - Use test databases/services

3. **Test Patterns**
   - Arrange-Act-Assert structure
   - Descriptive test names
   - Independent test cases

4. **Coverage Goals**
   - Critical paths: 100%
   - Overall: aim for 80%+
   - Edge cases and error handling',
'test', ARRAY['test', 'coverage', 'quality'], 'active', TRUE, 70.0),

('refactor', '1.0.0', 'Refactor code for better readability and maintainability',
'Refactor the specified code to improve:

1. **Readability**
   - Clear naming
   - Smaller functions
   - Consistent style

2. **Maintainability**
   - Single responsibility
   - Reduce coupling
   - Remove duplication

3. **Process**
   - Keep behavior unchanged
   - Make incremental changes
   - Run tests after each change
   - Document significant changes

Focus on making the code easier to understand and modify without changing its external behavior.',
'code', ARRAY['refactor', 'clean-code', 'quality'], 'active', TRUE, 70.0),

('debug', '1.0.0', 'Debug an issue or error in the codebase',
'Debug the reported issue:

1. **Understand the Problem**
   - Reproduce the issue
   - Gather error messages/logs
   - Identify expected vs actual behavior

2. **Investigate**
   - Add logging/debugging
   - Check recent changes
   - Review related code

3. **Fix**
   - Implement minimal fix
   - Add tests to prevent regression
   - Verify fix works

4. **Document**
   - Explain root cause
   - Note any related issues',
'code', ARRAY['debug', 'fix', 'troubleshoot'], 'active', TRUE, 75.0)

ON CONFLICT (name) DO NOTHING;

-- Verify
SELECT 'Skills:' as type, COUNT(*) as count FROM skills
UNION ALL
SELECT 'Commands:' as type, COUNT(*) as count FROM commands;
