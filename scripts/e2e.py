#!/usr/bin/env python3
"""End-to-end smoke test for homebox-mcp against a local Homebox instance.

Prerequisites:
  - Homebox v0.26+ running at http://localhost:7745 (e.g. docker container)
  - Binary built at ../homebox-mcp (run `go build -o homebox-mcp .` first)

The script registers a throwaway user (if needed), spawns the MCP server over
stdio and exercises every tool: read tools, create → update → attach → delete.
"""

import json
import os
import subprocess
import sys
import time
import urllib.request
import uuid

BASE = os.environ.get("HOMEBOX_URL", "http://localhost:7745")
EMAIL = f"e2e-{uuid.uuid4().hex[:8]}@example.com"
PASSWORD = "e2e-password-123"
HERE = os.path.dirname(os.path.abspath(__file__))
BINARY = os.path.join(HERE, "..", "homebox-mcp")


def http(method, path, payload=None, expect=(200, 201, 204)):
    url = BASE + path
    data = json.dumps(payload).encode() if payload is not None else None
    req = urllib.request.Request(url, data=data, method=method,
                                 headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return r.status, r.read().decode()
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        if e.code in expect:
            return e.code, body
        raise SystemExit(f"HTTP {method} {path} -> {e.code}: {body[:300]}")


def wait_for_homebox():
    for _ in range(30):
        try:
            s, _ = http("GET", "/api/v1/status")
            if s == 200:
                return
        except Exception:
            pass
        time.sleep(1)
    raise SystemExit("Homebox did not become ready at " + BASE)


class MCP:
    def __init__(self):
        env = dict(os.environ, HOMEBOX_URL=BASE, HOMEBOX_EMAIL=EMAIL,
                   HOMEBOX_PASSWORD=PASSWORD)
        self.proc = subprocess.Popen([BINARY], stdin=subprocess.PIPE,
                                     stdout=subprocess.PIPE, env=env,
                                     stderr=subprocess.DEVNULL, text=True)
        self.next_id = 0
        self._rpc("initialize", {"protocolVersion": "2025-06-18",
                                 "capabilities": {},
                                 "clientInfo": {"name": "e2e", "version": "0"}})
        self._notify("notifications/initialized")

    def _notify(self, method):
        self.proc.stdin.write(json.dumps({"jsonrpc": "2.0", "method": method}) + "\n")
        self.proc.stdin.flush()

    def _rpc(self, method, params):
        self.next_id += 1
        mid = self.next_id
        self.proc.stdin.write(json.dumps(
            {"jsonrpc": "2.0", "id": mid, "method": method, "params": params}) + "\n")
        self.proc.stdin.flush()
        while True:
            line = self.proc.stdout.readline()
            if not line:
                raise SystemExit("server closed stdout unexpectedly")
            msg = json.loads(line)
            if msg.get("id") == mid:
                if "error" in msg:
                    raise SystemExit(f"JSON-RPC error for {method}: {msg['error']}")
                return msg["result"]

    def call(self, tool, args):
        res = self._rpc("tools/call", {"name": tool, "arguments": args})
        if res.get("isError"):
            raise SystemExit(f"tool {tool} failed: {res['content'][0]['text'][:400]}")
        sc = res.get("structuredContent")
        if sc is not None:
            return sc
        return json.loads(res["content"][0]["text"])

    def close(self):
        self.proc.stdin.close()
        self.proc.wait(timeout=10)


def main():
    wait_for_homebox()
    status, body = http("POST", "/api/v1/users/register",
                        {"name": "E2E User", "email": EMAIL, "password": PASSWORD},
                        expect=(204,))
    print(f"registered user {EMAIL} (HTTP {status})")

    mcp = MCP()
    ok = lambda name, extra="": print(f"  PASS {name} {extra}")

    stats = mcp.call("get_stats", {})
    ok("get_stats", f"items={stats['statistics']['totalItems']}")

    types = mcp.call("list_entity_types", {})
    item_types = [t for t in types["entityTypes"] if not t["isLocation"]]
    assert item_types, "no item entity type found"
    ok("list_entity_types", f"{len(types['entityTypes'])} types")

    locs = mcp.call("list_locations", {})
    ok("list_locations", f"total={locs['total']}")

    tags = mcp.call("list_tags", {})
    ok("list_tags", f"total={tags['total']}")

    created = mcp.call("create_item", {
        "name": "E2E Test Drill",
        "description": "created by e2e.py",
        "quantity": 2,
    })
    assert created["created"], created
    item_id = created["item"]["id"]
    ok("create_item", f"id={item_id}")

    updated = mcp.call("update_item", {
        "id": item_id,
        "purchaseDate": "2026-03-01",
        "purchaseFrom": "E2E Store",
        "purchasePrice": 8500,
        "notes": "updated by e2e",
        "insured": True,
    })
    assert updated["updated"]
    it = updated["item"]
    assert it["purchasePrice"] == 8500, it["purchasePrice"]
    assert it["purchaseDate"].startswith("2026-03-01"), it["purchaseDate"]
    assert it["insured"] is True and it["quantity"] == 2
    ok("update_item", f"price={it['purchasePrice']} date={it['purchaseDate']} qty kept={it['quantity']}")

    png = "/tmp/e2e-attachment.png"
    with open(png, "wb") as f:
        # 1x1 red pixel PNG
        f.write(bytes.fromhex(
            "89504e470d0a1a0a0000000d4948445200000001000000010802000000907753"
            "de0000000c4944415408d763f8cfc0f01f0005050200f2b4a1c90000000049454e"
            "44ae426082"))
    att = mcp.call("add_attachment", {"itemId": item_id, "filePath": png,
                                      "type": "photo", "primary": True})
    assert att["attached"] and att["itemAttachments"], att
    primary = [a for a in att["itemAttachments"] if a["primary"]]
    assert primary, f"no primary attachment: {att['itemAttachments']}"
    ok("add_attachment", f"{len(att['itemAttachments'])} attachment(s), primary set")

    full = mcp.call("get_item", {"id": item_id})
    assert full["item"]["name"] == "E2E Test Drill"
    ok("get_item")

    search = mcp.call("search_items", {"q": "E2E Test Drill"})
    assert any(i["id"] == item_id for i in search["items"]), search
    ok("search_items", f"total={search['total']}")

    asset_id = full["item"].get("assetId", "")
    if asset_id:
        by_asset = mcp.call("get_item_by_asset_id", {"assetId": asset_id})
        assert by_asset["found"], by_asset
        ok("get_item_by_asset_id", f"assetId={asset_id}")
    else:
        print("  SKIP get_item_by_asset_id (item has no asset ID assigned)")

    deleted = mcp.call("delete_item", {"id": item_id})
    assert deleted["deleted"]
    ok("delete_item", f"id={item_id}")

    mcp.close()
    print("ALL E2E TESTS PASSED")


if __name__ == "__main__":
    main()
