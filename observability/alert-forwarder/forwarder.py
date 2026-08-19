#!/usr/bin/env python3
"""Alertmanager webhook forwarder: jsonl persist + local mail + external mail + webhook fanout.
webhooks.list: one URL per line; suffix #dingtalk/#wecom/#feishu wraps bot JSON."""
import json, smtplib, socket, sys, urllib.request
from datetime import datetime, timezone
from email.message import EmailMessage
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

LOG = "/var/log/alert-forwarder/alerts.jsonl"
WH_LIST = "/etc/alert-forwarder/webhooks.list"
CFG = "/etc/alert-forwarder/config.env"
HOSTNAME = socket.gethostname()
NL = chr(10)

def load_cfg():
    cfg = {}
    try:
        for line in open(CFG):
            line = line.strip()
            if line and not line.startswith("#") and "=" in line:
                k, v = line.split("=", 1)
                cfg[k.strip()] = v.strip()
    except FileNotFoundError:
        pass
    return cfg

def summarize(data):
    alerts = data.get("alerts", [])
    firing = [a for a in alerts if a.get("status") == "firing"]
    resolved = [a for a in alerts if a.get("status") == "resolved"]
    lines = []
    for a in firing:
        lb = a.get("labels", {})
        lines.append("[FIRING] %s %s" % (lb.get("alertname", "?"), lb.get("instance", "")))
    for a in resolved:
        lb = a.get("labels", {})
        lines.append("[RESOLVED] %s %s" % (lb.get("alertname", "?"), lb.get("instance", "")))
    head = "yuxin-gateway@%s: %d firing / %d resolved" % (HOSTNAME, len(firing), len(resolved))
    return head, NL.join(lines) or "(no alerts)"

def send_mail(subject, body, cfg):
    recipients = ["root@%s" % HOSTNAME]
    ext = cfg.get("ALERT_EMAIL_EXTERNAL", "")
    if ext:
        recipients += [x.strip() for x in ext.split(",") if x.strip()]
    for rcpt in recipients:
        try:
            msg = EmailMessage()
            msg["From"] = cfg.get("MAIL_FROM", "alert-forwarder@%s" % HOSTNAME)
            msg["To"] = rcpt
            msg["Subject"] = subject
            msg.set_content(body)
            with smtplib.SMTP("127.0.0.1", 25, timeout=10) as s:
                s.send_message(msg)
        except Exception as e:
            print("mail->%s failed: %s" % (rcpt, e), file=sys.stderr)

def fanout_webhooks(data, text):
    try:
        urls = [l.strip() for l in open(WH_LIST) if l.strip() and not l.startswith("#")]
    except FileNotFoundError:
        return
    for entry in urls:
        url, _, fmt = entry.partition("#")
        url, fmt = url.strip(), fmt.strip()
        try:
            if fmt == "dingtalk":
                payload = {"msgtype": "markdown", "markdown": {"title": "yuxin alert", "text": text}}
            elif fmt in ("wecom", "wechat-work"):
                payload = {"msgtype": "text", "text": {"content": text}}
            elif fmt == "feishu":
                payload = {"msg_type": "text", "content": {"text": text}}
            else:
                payload = data
            req = urllib.request.Request(url, data=json.dumps(payload).encode(),
                                         headers={"Content-Type": "application/json"})
            urllib.request.urlopen(req, timeout=10).read()
        except Exception as e:
            print("webhook->%s failed: %s" % (url, e), file=sys.stderr)

class H(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(200); self.end_headers(); self.wfile.write(b"ok")
        else:
            self.send_response(404); self.end_headers()
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        raw = self.rfile.read(n) if n else b"{}"
        try:
            data = json.loads(raw)
        except Exception:
            data = {"raw": raw.decode(errors="replace")}
        head, body = summarize(data)
        ts = datetime.now(timezone.utc).isoformat()
        with open(LOG, "a") as f:
            f.write(json.dumps({"ts": ts, "data": data}, ensure_ascii=False) + NL)
        cfg = load_cfg()
        text = head + NL + body + NL + NL + json.dumps(data, ensure_ascii=False, indent=1)[:4000]
        send_mail(head, text, cfg)
        fanout_webhooks(data, text)
        self.send_response(200); self.end_headers(); self.wfile.write(b"accepted")
    def log_message(self, *a):
        pass

if __name__ == "__main__":
    srv = ThreadingHTTPServer(("0.0.0.0", 5001), H)
    print("alert-forwarder listening on 127.0.0.1:5001")
    srv.serve_forever()
