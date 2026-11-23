#!/usr/bin/env python3
"""Download the scene currently active at a parcel from a Catalyst.

Writes every file of the entity, plus its scene.json, to OUT_DIR, so the
deployer can publish the same content to another Catalyst. Used to snapshot
campusFGA, whose deployed version no longer exists as source anywhere.

    python3 export-scene.py http://localhost/content 0,0 ../scenes/campusFGA \
        --exclude 'Claude outputs/'
"""
import argparse
import json
import pathlib
import urllib.request


def fetch(url, data=None):
    headers = {"Content-Type": "application/json"} if data else {}
    req = urllib.request.Request(url, data=data, headers=headers)
    with urllib.request.urlopen(req, timeout=60) as resp:
        return resp.read()


def main():
    p = argparse.ArgumentParser(description="Download the scene active at a parcel from a Catalyst.")
    p.add_argument("content_url", help="Catalyst content URL, e.g. http://localhost/content")
    p.add_argument("pointer", help="any parcel of the scene, e.g. 0,0")
    p.add_argument("out_dir")
    p.add_argument("--exclude", action="append", default=[], help="skip files starting with this prefix")
    args = p.parse_args()

    base = args.content_url.rstrip("/")
    body = json.dumps({"pointers": [args.pointer]}).encode()
    entities = json.loads(fetch(f"{base}/entities/active", body))
    if not entities:
        raise SystemExit(f"no active entity at {args.pointer}")
    entity = entities[0]

    out = pathlib.Path(args.out_dir)
    for item in entity["content"]:
        name = item["file"]
        if any(name.startswith(prefix) for prefix in args.exclude):
            print(f"skip {name}")
            continue
        target = out / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(fetch(f"{base}/contents/{item['hash']}"))

    # The entity metadata is scene.json plus what the SDK added when deploying
    # (sdkVersion). The deployer republishes it as is; dotfiles are not
    # published as scene content.
    (out / ".entity-metadata.json").write_text(json.dumps(entity["metadata"], indent=2) + "\n")
    print(f"exported {entity['id']} ({entity['metadata']['display']['title']}) to {out}")


if __name__ == "__main__":
    main()
