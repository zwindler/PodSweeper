# Player Terminal

The player terminal is a pod that provides a restricted environment for playing PodSweeper. It uses the `player` ServiceAccount with limited RBAC permissions, enforcing the CTF rules based on the current level.

## Joining the Game

```bash
# Option 1: Using make
make play

# Option 2: Using kubectl directly
kubectl exec -it player -n podsweeper-game -- bash
```

## Available Commands

Inside the terminal, you have access to `kubectl` and several convenience aliases:

| Command | Description |
|---------|-------------|
| `pods` | List all pods in the game namespace |
| `grid` | Show only cell pods (the game grid) |
| `hints` | Show only hint pods |
| `click pod-X-Y` | Delete a pod (equivalent to clicking a cell) |
| `sweep X Y` | Helper to click cell at coordinates X, Y |
| `hint X Y` | Get the hint value for a revealed cell |
| `map` | Try to read the mine map (works on Level 0-1) |
| `level` | Show current game level |
| `status` | Show game status from ConfigMap |

## Example Session

```bash
$ make play
Joining PodSweeper as player...

    ____            ______                                  
   / __ \____  ____/ / ___/      _____  ___  ____  ___  _____
  / /_/ / __ \/ __  /\__ \ | /| / / _ \/ _ \/ __ \/ _ \/ ___/
 / ____/ /_/ / /_/ /___/ / |/ |/ /  __/  __/ /_/ /  __/ /    
/_/    \____/\__,_//____/|__/|__/\___/\___/ .___/\___/_/     
                                         /_/                 

Welcome to PodSweeper - The Kubernetes Minesweeper CTF!
...

podsweeper> pods
NAME         READY   STATUS    RESTARTS   AGE
pod-0-0      1/1     Running   0          1m
pod-0-1      1/1     Running   0          1m
...

podsweeper> sweep 2 2
pod "pod-2-2" deleted

podsweeper> pods
NAME         READY   STATUS    RESTARTS   AGE
hint-2-2     1/1     Running   0          5s
pod-0-0      1/1     Running   0          1m
...

podsweeper> hint 2 2
1
```

## RBAC Restrictions by Level

The player's permissions change as levels progress:

| Level | Available Permissions | Cheat Path |
|-------|----------------------|------------|
| 0 | Full access | Read `map` ConfigMap directly |
| 1 | No ConfigMap access | Decode `map` Secret (Base64) |
| 2 | No Secret access | Exec into game pods, read env vars |
| 3 | No exec on game pods | Exec into gamemaster, read /tmp/map.txt |
| 4+ | Minimal (delete pods only) | No cheating possible |

## Customizing the Terminal Image

You can use your own terminal image with additional tools. This is useful for:
- Adding your favorite shell (zsh, fish)
- Including debugging tools
- Pre-installing scripts for automation challenges

### Option 1: Override via Kustomize

Create a kustomization overlay:

```yaml
# deploy/custom/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../base

images:
  - name: ghcr.io/zwindler/podsweeper-player-terminal
    newName: your-registry/your-terminal
    newTag: latest
```

Deploy with:
```bash
kubectl apply -k deploy/custom
```

### Option 2: Build a Custom Image

Use the default image as a base:

```dockerfile
# Dockerfile.custom-terminal
FROM ghcr.io/zwindler/podsweeper-player-terminal:latest

# Add your tools
RUN apk add --no-cache \
    vim \
    tmux \
    zsh \
    fzf

# Add custom scripts
COPY my-scripts/ /usr/local/bin/

# Switch to zsh if preferred
# CMD ["zsh"]
```

Build and use:
```bash
podman build -t my-terminal:latest -f Dockerfile.custom-terminal .
# Load into kind or push to registry
```

### Option 3: Direct Pod Edit

For quick testing, edit the player pod directly:

```bash
kubectl delete pod player -n podsweeper-game
kubectl run player -n podsweeper-game \
  --image=your-image:tag \
  --serviceaccount=player \
  --command -- sleep infinity
```

## Troubleshooting

### "Player pod not found"

The player pod uses `restartPolicy: Never`, so it won't auto-restart after deletion or cluster restart:

```bash
kubectl apply -f deploy/base/player-terminal.yaml
# or
make play  # This will auto-create the pod
```

### "Permission denied" errors

This is expected behavior for higher levels. Check your current level:

```bash
podsweeper> level
2
podsweeper> kubectl get secrets
Error from server (Forbidden): secrets is forbidden...
```

The Gamemaster updates RBAC rules based on the level. You need to find another way!

### Latency issues (Level 8)

Level 8 requires precise timing. Running from inside the cluster (player pod) provides lower latency than external kubectl commands. If you're still having issues:

```bash
# Check latency to API server
time kubectl get pods > /dev/null

# Should be < 50ms from inside the cluster
```
