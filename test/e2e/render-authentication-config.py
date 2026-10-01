"""Render deploy/apiserver/authentication-config.yaml for the fixture.

The fixture's apiserver reads the same file the install guide tells people to
use, with this environment's issuer, client and self-signed CA, and v1beta1
because the fixture runs Kubernetes 1.31.

    python3 render-authentication-config.py CA_FILE OUT_FILE
"""
import sys

import yaml

ca_file, out_file = sys.argv[1:3]
with open("deploy/apiserver/authentication-config.yaml") as f:
    config = yaml.safe_load(f)
config["apiVersion"] = "apiserver.config.k8s.io/v1beta1"
(jwt,) = config["jwt"]
jwt["issuer"]["url"] = "https://login.room-pass.test:18443"
# Every fixture client that sends its token to Kubernetes. A token whose aud is
# absent here is rejected -- and the symptom is only "Unauthorized" in the
# application's log while login itself still succeeds.
jwt["issuer"]["audiences"] = ["room-pass-demo"]
with open(ca_file) as f:
    jwt["issuer"]["certificateAuthority"] = f.read()
with open(out_file, "w") as f:
    yaml.safe_dump(config, f, sort_keys=False)
