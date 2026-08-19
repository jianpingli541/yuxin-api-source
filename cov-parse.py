import re, pathlib, sys

lines = pathlib.Path("/tmp/test-result.txt").read_text().splitlines()
rows = []
for ln in lines:
    m = re.search(r"^ok\s+(\S+)\s+[\d.]+s\s+coverage:\s+([\d.]+)%", ln)
    if m:
        rows.append((float(m.group(2)), m.group(1)))
    else:
        m2 = re.search(r"^ok\s+(\S+)\s+[\d.]+s", ln)
        if m2:
            rows.append((None, m2.group(1)))

rows.sort(key=lambda x: (x[0] if x[0] is not None else -1))
print("=== Lowest coverage packages ===")
for cov, p in rows[:35]:
    covs = "  no tests" if cov is None else f"{cov:5.1f}%"
    print(f"  {covs:>12}  {p}")
print()
print("Total packages with tests:", len(rows))
covs = [c for c, _ in rows if c is not None]
print(f"Current overall (avg of reported): {sum(covs)/len(covs):.1f}%")
