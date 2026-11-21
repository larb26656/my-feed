#!/usr/bin/env python3
"""Translate the generator's nullable extension into OpenAPI 3.1 types."""
import json
import sys

path = sys.argv[1]
with open(path, encoding="utf-8") as source:
    document = json.load(source)


def visit(value):
    if isinstance(value, dict):
        if value.pop("x-nullable", False):
            value["type"] = [value["type"], "null"]
        for child in value.values():
            visit(child)
    elif isinstance(value, list):
        for child in value:
            visit(child)


visit(document)
with open(path, "w", encoding="utf-8") as output:
    json.dump(document, output, indent=4, ensure_ascii=False)
    output.write("\n")
