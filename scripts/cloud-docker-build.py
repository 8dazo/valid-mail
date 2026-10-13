"""Build through an injected HTTPS proxy without retaining proxy credentials."""

import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
from urllib.parse import urlsplit


def main():
    command = ["docker", "build", "--tag", sys.argv[1] if len(sys.argv) > 1 else "weme:local"]
    proxy = os.environ.get("HTTPS_PROXY") or os.environ.get("https_proxy")
    if proxy:
        host = urlsplit(proxy).hostname
        if not host:
            raise SystemExit("The injected HTTPS proxy has no hostname.")
        command += ["--add-host", f"{host}:{socket.gethostbyname(host)}"]
        for name in ["HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"]:
            if name in os.environ:
                command += ["--build-arg", name]
        trust_bundle = os.environ.get("SSL_CERT_FILE")
        if trust_bundle and os.path.isfile(trust_bundle):
            command += ["--secret", f"id=ca-certificates,src={trust_bundle}"]
    command += ["."]
    with tempfile.TemporaryDirectory(prefix="weme-docker-", dir="/tmp") as docker_config:
        environment = os.environ.copy()
        environment.setdefault("DOCKER_CONFIG", docker_config)
        # Predefined Docker proxy arguments are excluded from image history.
        # The public CA bundle is a transient BuildKit mount, never a COPY.
        return subprocess.call(command, cwd=Path(__file__).resolve().parents[1], env=environment)


if __name__ == "__main__":
    raise SystemExit(main())
