#!/usr/bin/env bash
# Crée l'arborescence du lab Argo CD sur k3s.
# Idempotent : un fichier qui existe déjà n'est jamais écrasé.
# Usage : ./scripts/create_tree.sh [dossier_cible]   (par défaut : dossier courant)
set -euo pipefail

ROOT="${1:-.}"
mkdir -p "$ROOT"
cd "$ROOT"

# write_if_missing CHEMIN : lit le contenu sur stdin, ne l'écrit que si le fichier n'existe pas
write_if_missing() {
  local path="$1"
  if [ -e "$path" ]; then
    echo "existe déjà : $path"
    cat > /dev/null
    return 0
  fi
  mkdir -p "$(dirname "$path")"
  cat > "$path"
  echo "créé        : $path"
}

mkdir -p app deploy/base deploy/overlays/dev argocd crds docs/captures .github/workflows scripts

write_if_missing README.md <<'EOF'
# Lab Argo CD sur k3s : du déploiement GitOps au rollback

Statut : en cours. Compléter chaque étape au fil du lab.
EOF

write_if_missing .gitignore <<'EOF'
# Secrets et configuration locale : ne jamais committer
.env
*.env
kubeconfig
*.kubeconfig
*.pem
*.key
secrets/
*secret*.yaml

# Python
__pycache__/
*.pyc

# Captures brutes (les captures publiables vont dans docs/captures/)
docs/captures/brut/
EOF

write_if_missing app/main.py <<'EOF'
"""Petite app « hello » : renvoie un message en JSON.

Le message vient de MESSAGE (typiquement un ConfigMap), le port de PORT (8080 par défaut).
"""
import json
import os
from http.server import BaseHTTPRequestHandler, HTTPServer

MESSAGE = os.environ.get("MESSAGE", "Hello from Python on Kubernetes!")
PORT = int(os.environ.get("PORT", "8080"))


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = {"status": "ok"} if self.path == "/healthz" else {"message": MESSAGE}
        payload = json.dumps(body, ensure_ascii=False).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def log_message(self, fmt, *args):
        print(f"{self.address_string()} {fmt % args}", flush=True)


if __name__ == "__main__":
    print(f"listening on :{PORT}", flush=True)
    HTTPServer(("0.0.0.0", PORT), Handler).serve_forever()
EOF

write_if_missing app/Dockerfile <<'EOF'
# Étape 5 bis : image de l'app (à compléter).
# Étape 9 : utilisateur non root, port 8080, CMD.
EOF

write_if_missing deploy/base/deployment.yaml <<'EOF'
# Étape 5 : Deployment de l'app (image, port 8080, MESSAGE depuis le ConfigMap).
# Étape 7 : montage du PVC. Étape 9 : durcissement.
EOF

write_if_missing deploy/base/service.yaml <<'EOF'
# Étape 5 : Service NodePort vers le port 8080 du pod.
EOF

write_if_missing deploy/base/configmap.yaml <<'EOF'
# Étape 5 : ConfigMap avec la clé MESSAGE. Changer le message = nouvelle version (étape 6).
EOF

write_if_missing deploy/base/pvc.yaml <<'EOF'
# Étape 7 : PersistentVolumeClaim (StorageClass local-path, par défaut sur k3s).
EOF

write_if_missing deploy/overlays/dev/kustomization.yaml <<'EOF'
# Étapes 5 bis et 10 : tag d'image (SHA du commit) pour l'environnement dev.
EOF

write_if_missing argocd/application.yaml <<'EOF'
# Étapes 3 et 5 : Application Argo CD pointant vers deploy/.
# Étape 4 : syncPolicy (selfHeal, prune).
EOF

write_if_missing crds/greeting-crd.yaml <<'EOF'
# Étape 8 : CustomResourceDefinition « Greeting » (champ message).
EOF

write_if_missing crds/greeting-example.yaml <<'EOF'
# Étape 8 : une ressource Greeting, mise à jour par commit.
EOF

write_if_missing docs/captures/.gitkeep <<'EOF'
EOF

write_if_missing .github/workflows/build.yaml <<'EOF'
# Étape 10 : CI — tests, build, push sur Docker Hub, mise à jour du tag (SHA) dans deploy/.
EOF

echo "Arborescence prête dans : $(pwd)"
