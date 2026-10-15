"""Assert the seven demo results; uses only the Python standard library."""
import json
import sys

with open(sys.argv[1], encoding="utf-8") as source:
    results = json.load(source)
with open("examples/input.json", encoding="utf-8") as source:
    inputs = json.load(source)
expected = [
    ("SSH", "OpenSSH", "8.9p1", "Ubuntu"),
    ("HTTP", "nginx", "1.24.0", ""),
    ("HTTP", "Apache httpd", "2.4.57", "Debian"),
    ("MySQL", "MySQL", "8.0.36", ""),
    ("Redis", "Redis", "7.2.4", ""),
    ("FTP", "ProFTPD", "1.3.8", ""),
    ("HTTP", "Jetty", "9.4.54.v20240208", ""),
]
assert len(results) == len(expected), results
for item, original, want in zip(results, inputs, expected):
    actual = tuple(item[key] for key in ("protocol", "product", "version", "os_hint"))
    assert actual == want, (actual, want)
    assert (item["ip"], item["port"]) == (original["ip"], original["port"]), item
    assert item["confidence"] == 0.95, item
print("All seven demo results verified.")
