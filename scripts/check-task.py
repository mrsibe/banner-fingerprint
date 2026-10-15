"""Compare real Client output against all 20 independently maintained task cases."""
import json
import sys

with open("examples/task-expected.json", encoding="utf-8") as source:
    expected = json.load(source)
with open(sys.argv[1], encoding="utf-8") as source:
    actual = json.load(source)
assert len(actual) == len(expected) == 20
for index, (got, want) in enumerate(zip(actual, expected)):
    assert got == want, (index, got, want)
print("All 20 task samples verified (18 recognized, 2 unknown).")
