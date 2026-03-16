#!/usr/bin/env python3
import sys, csv, html

rows = list(csv.reader(sys.stdin))
if not rows:
    sys.exit(0)

header = rows[0]
body = rows[1:]

def cell(content, tag):
    return '<' + tag + '>' + html.escape(content) + '</' + tag + '>'

out = '<style>'
out += '.csv-wrap{font-family:monospace;font-size:13px;color:#ccc;overflow-x:auto}'
out += '.csv-wrap table{border-collapse:collapse}'
out += '.csv-wrap th{color:#666;text-align:left;padding:2px 16px 6px 0;border-bottom:1px solid #222}'
out += '.csv-wrap td{padding:2px 16px 2px 0;color:#aaa}'
out += '.csv-wrap tr:hover td{color:#eee}'
out += '</style>'
out += '<div class="csv-wrap"><table><thead><tr>'
for h in header:
    out += cell(h, 'th')
out += '</tr></thead><tbody>'
for row in body:
    out += '<tr>'
    for col in row:
        out += cell(col, 'td')
    out += '</tr>'
out += '</tbody></table></div>'

sys.stdout.write(out)
sys.stdout.flush()
