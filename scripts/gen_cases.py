#!/usr/bin/env python3
"""Deterministically generate the 200,000 differential-test cases.

Usage: scripts/gen_cases.py > cases.tsv
Protocol (tab-separated), see internal/difftest/main.go:
  D <id-hex> <decimal>   A <id-hex> <value>   X <hex>
"""
import random
import sys

random.seed(20261018)
ids = ["42", "43", "44", "45", "46", "47"]
ids_all = ids + ["ffff", "0", "41", "48", "1234"]
rows = []


def dec():
    r = random.random()
    if r < 0.15:
        return random.choice(["", "+", "-", ".", "+.", "-.5", "1.", "1.2.3", "1e2", "1E-3", "0x10", " 1", "1 ",
                              "\u0661\u0662", "١٢", "--1", "+-1", "1,5", "NaN", "inf", "0..1", "00", "-0", "+0",
                              "000123.4500", "0.000", "-0.0005", "0.0005"])
    s = random.choice(["", "", "-", "+"])
    w = random.choice([0, 1, 1, 1, 2, 3, 5, 6, 8, 9, 10, 12, 20, 37, 38, 39, 40, 41, 60])
    whole = "".join(random.choice("0123456789") for _ in range(w)) if w else random.choice(["0", "1"])
    f = random.choice([0, 0, 1, 2, 3, 3, 4, 4, 5, 6, 9, 17, 18, 19, 20])
    if r < 0.5 and f >= 4:  # near ties
        frac = "".join(random.choice("0123456789") for _ in range(f - 1))
        frac = frac[:3].ljust(3, "0") + random.choice("0045") + "5" * 0
        f = len(frac)
    else:
        frac = "".join(random.choice("0123456789") for _ in range(f))
    return s + whole + (("." + frac) if f else "")


for _ in range(120000):
    rows.append("D\t%s\t%s" % (random.choice(ids_all if random.random() < 0.1 else ids), dec()))
bounds = [-2**31, -2**31 + 1, -80001, -80000, -1, 0, 1, 22500, 99999, 100000, 100001, 150000, 150001, 200000,
          200001, 100000000, 100000001, 1000000, 1000001, 2**31 - 1]
for _ in range(40000):
    v = random.choice(bounds) if random.random() < 0.6 else random.randint(-2**31, 2**31 - 1)
    rows.append("A\t%s\t%d" % (random.choice(ids_all), v))
for _ in range(40000):
    n = random.choice([0, 1, 5, 6, 6, 6, 6, 7, 8])
    h = "".join(random.choice("0123456789abcdef") for _ in range(2 * n))
    if n == 6 and random.random() < 0.7:
        h = random.choice(ids)
        h = "00" + h + "%08x" % (random.randint(0, 2**32 - 1))
    rows.append("X\t%s" % h)
random.shuffle(rows)
sys.stdout.write("\n".join(rows) + "\n")
