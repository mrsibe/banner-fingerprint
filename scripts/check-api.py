"""Exercise canonical/legacy routes and unknown/error recovery on a running server."""
import json
import sys
import urllib.error
import urllib.request

base = sys.argv[1].rstrip("/") if len(sys.argv) > 1 else "http://localhost:8080"


def request(path, payload=None):
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(
        base + path, data=data, headers={"Content-Type": "application/json"}
    )
    with urllib.request.urlopen(req, timeout=10) as response:
        return json.load(response)


inputs = [
    {"ip": "192.0.2.1", "port": 80, "banner": "SSH-2.0-OpenSSH_8.9p1"},
    {"ip": "192.0.2.2", "port": 22, "banner": "not a known banner"},
    {"ip": "192.0.2.3", "port": 3306, "banner": ""},
    {"ip": "192.0.2.4", "port": 6379},
    {"ip": "192.0.2.5", "port": 80, "banner_base64": "AAECAw=="},
]
results = request("/fingerprint", inputs)
assert results == request("/api/v1/identify", inputs)
assert len(results) == len(inputs)
assert results[0]["protocol"] == "SSH" and results[0]["version"] == "8.9p1"
keys = {"ip", "port", "protocol", "product", "version", "os_hint", "confidence"}
for index, item in enumerate(results):
    assert set(item) == keys and 0 <= item["confidence"] <= 1, item
    assert (item["ip"], item["port"]) == (inputs[index]["ip"], inputs[index]["port"])
    if index:
        assert item["protocol"] == "unknown" and item["confidence"] == 0, item
        assert all(item[key] == "" for key in ("product", "version", "os_hint")), item
try:
    request("/fingerprint", [{"ip": "invalid", "port": 22, "banner": "x"}])
except urllib.error.HTTPError as error:
    assert error.code == 400
    assert json.load(error)["error"]
else:
    raise AssertionError("invalid IP was accepted")
for path in ("/health", "/healthz"):
    assert request(path) == {"status": "ok"}
print("Canonical routes, compatibility routes, unknown results and recovery verified.")
