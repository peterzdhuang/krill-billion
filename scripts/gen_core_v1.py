#!/usr/bin/env python3
# The committed packs/core-v1.json comes from `make packbuild`.
"""Hand-build packs/core-v1.json when a full live packbuild is too slow."""

from __future__ import annotations

import json
from datetime import datetime, timezone


def e(qid: str, label: str, aliases: list[str], rarity: int) -> dict:
    return {"id": qid, "label": label, "aliases": aliases, "rarity": rarity}


def prompt(pid: str, category: str, text: str, entries: list[dict]) -> dict:
    entries = sorted(entries, key=lambda x: (x["rarity"], x["id"]))
    return {"id": pid, "category": category, "text": text, "entries": entries}


def main() -> None:
    prompts: list[dict] = []

    # ——— Countries ———
    prompts.append(
        prompt(
            "country",
            "Countries",
            "A country",
            [
                e("Q30", "United States", ["USA", "US", "America"], 0),
                e("Q145", "United Kingdom", ["UK", "Britain", "Great Britain"], 2),
                e("Q142", "France", [], 4),
                e("Q183", "Germany", [], 6),
                e("Q17", "Japan", [], 8),
                e("Q155", "Brazil", [], 12),
                e("Q408", "Australia", [], 15),
                e("Q668", "India", [], 18),
                e("Q159", "Russia", [], 22),
                e("Q148", "China", ["PRC"], 25),
                e("Q889", "Afghanistan", [], 45),
                e("Q1005", "The Gambia", ["Gambia"], 72),
                e("Q23681", "Nauru", [], 92),
                e("Q695", "Palau", [], 95),
            ],
        )
    )
    prompts.append(
        prompt(
            "landlocked-country",
            "Countries",
            "A landlocked country",
            [
                e("Q39", "Switzerland", [], 5),
                e("Q40", "Austria", [], 10),
                e("Q28", "Hungary", [], 18),
                e("Q213", "Czech Republic", ["Czechia"], 22),
                e("Q889", "Afghanistan", [], 32),
                e("Q750", "Bolivia", [], 40),
                e("Q965", "Burkina Faso", [], 55),
                e("Q1036", "Uganda", [], 62),
                e("Q917", "Bhutan", [], 78),
                e("Q819", "Laos", ["Lao PDR"], 88),
            ],
        )
    )
    prompts.append(
        prompt(
            "african-country",
            "Countries",
            "A country in Africa",
            [
                e("Q258", "South Africa", [], 5),
                e("Q1033", "Nigeria", [], 10),
                e("Q1028", "Morocco", [], 15),
                e("Q1016", "Egypt", [], 18),
                e("Q1049", "Senegal", [], 35),
                e("Q1009", "Cameroon", [], 42),
                e("Q1005", "The Gambia", ["Gambia"], 70),
                e("Q912", "Mali", [], 75),
                e("Q962", "Benin", [], 82),
                e("Q971", "Republic of the Congo", ["Congo-Brazzaville"], 90),
            ],
        )
    )
    prompts.append(
        prompt(
            "island-country",
            "Countries",
            "An island country",
            [
                e("Q17", "Japan", [], 4),
                e("Q664", "New Zealand", [], 12),
                e("Q334", "Singapore", [], 18),
                e("Q252", "Indonesia", [], 22),
                e("Q233", "Malta", [], 40),
                e("Q23681", "Nauru", [], 78),
                e("Q695", "Palau", [], 85),
                e("Q678", "Tonga", [], 90),
                e("Q697", "Tuvalu", [], 95),
            ],
        )
    )
    prompts.append(
        prompt(
            "european-country",
            "Countries",
            "A country in Europe",
            [
                e("Q142", "France", [], 3),
                e("Q183", "Germany", [], 5),
                e("Q145", "United Kingdom", ["UK", "Britain"], 7),
                e("Q38", "Italy", [], 10),
                e("Q29", "Spain", [], 12),
                e("Q55", "Netherlands", ["Holland"], 20),
                e("Q224", "Croatia", [], 45),
                e("Q236", "Montenegro", [], 72),
                e("Q221", "North Macedonia", ["Macedonia"], 85),
                e("Q233", "Malta", [], 90),
            ],
        )
    )

    # ——— Capitals ———
    prompts.append(
        prompt(
            "national-capital",
            "National capitals",
            "A national capital",
            [
                e("Q84", "London", [], 2),
                e("Q90", "Paris", [], 4),
                e("Q64", "Berlin", [], 8),
                e("Q649", "Moscow", [], 12),
                e("Q1486", "Buenos Aires", [], 22),
                e("Q19660", "Bucharest", [], 48),
                e("Q1863", "Islamabad", [], 55),
                e("Q3881", "Paramaribo", [], 82),
                e("Q9270", "Suva", [], 90),
                e("Q131694", "Ngerulmud", [], 98),
            ],
        )
    )
    prompts.append(
        prompt(
            "european-capital",
            "National capitals",
            "A European capital",
            [
                e("Q84", "London", [], 2),
                e("Q90", "Paris", [], 4),
                e("Q64", "Berlin", [], 8),
                e("Q220", "Rome", [], 12),
                e("Q2807", "Madrid", [], 15),
                e("Q1741", "Vienna", [], 25),
                e("Q19660", "Bucharest", [], 50),
                e("Q1773", "Riga", [], 70),
                e("Q1770", "Tallinn", [], 78),
                e("Q3901", "Podgorica", [], 92),
            ],
        )
    )
    prompts.append(
        prompt(
            "asian-capital",
            "National capitals",
            "A capital in Asia",
            [
                e("Q1490", "Tokyo", [], 3),
                e("Q956", "Beijing", [], 8),
                e("Q987", "New Delhi", [], 12),
                e("Q1865", "Kuala Lumpur", [], 28),
                e("Q1863", "Islamabad", [], 40),
                e("Q1858", "Hanoi", [], 45),
                e("Q23430", "Ulaanbaatar", [], 72),
                e("Q200342", "Naypyidaw", [], 85),
                e("Q3820", "Thimphu", [], 92),
                e("Q927", "Dili", [], 96),
            ],
        )
    )
    prompts.append(
        prompt(
            "african-capital",
            "National capitals",
            "An African capital",
            [
                e("Q85", "Cairo", [], 5),
                e("Q34647", "Pretoria", [], 18),
                e("Q3659", "Nairobi", [], 25),
                e("Q1960", "Accra", [], 30),
                e("Q3718", "Addis Ababa", [], 40),
                e("Q3739", "Luanda", [], 55),
                e("Q3832", "Lilongwe", [], 75),
                e("Q3777", "Gaborone", [], 82),
                e("Q3905", "Mbabane", [], 90),
                e("Q3932", "Moroni", [], 95),
            ],
        )
    )
    prompts.append(
        prompt(
            "south-american-capital",
            "National capitals",
            "A South American capital",
            [
                e("Q1486", "Buenos Aires", [], 5),
                e("Q2844", "Brasília", ["Brasilia"], 10),
                e("Q2887", "Santiago", [], 18),
                e("Q3630", "Lima", [], 22),
                e("Q34863", "Bogotá", ["Bogota"], 28),
                e("Q3551", "Caracas", [], 40),
                e("Q3624", "Montevideo", [], 55),
                e("Q2907", "Asunción", ["Asuncion"], 65),
                e("Q3915", "Georgetown", [], 80),
                e("Q3881", "Paramaribo", [], 88),
            ],
        )
    )

    # ——— Dog breeds ———
    prompts.append(
        prompt(
            "dog-breed",
            "Dog breeds",
            "A dog breed",
            [
                e("Q38904", "Labrador Retriever", ["Labrador", "Lab"], 3),
                e("Q38690", "Golden Retriever", ["Golden"], 5),
                e("Q38604", "German Shepherd", ["Alsatian"], 7),
                e("Q38404", "Bulldog", ["English Bulldog"], 12),
                e("Q37983", "Poodle", [], 15),
                e("Q38386", "Beagle", [], 18),
                e("Q38602", "Dachshund", ["Wiener dog", "Sausage dog"], 28),
                e("Q39107", "Shiba Inu", ["Shiba"], 40),
                e("Q38870", "Basenji", [], 78),
                e("Q39052", "Xoloitzcuintle", ["Xolo", "Mexican Hairless"], 92),
            ],
        )
    )
    prompts.append(
        prompt(
            "terrier",
            "Dog breeds",
            "A terrier",
            [
                e("Q38180", "Yorkshire Terrier", ["Yorkie"], 5),
                e("Q38278", "Bull Terrier", [], 15),
                e("Q38315", "Scottish Terrier", ["Scottie"], 25),
                e("Q38348", "Airedale Terrier", ["Airedale"], 35),
                e("Q38440", "Fox Terrier", [], 42),
                e("Q38372", "Cairn Terrier", [], 48),
                e("Q38412", "Border Terrier", [], 55),
                e("Q38510", "Bedlington Terrier", [], 72),
                e("Q38555", "Skye Terrier", [], 85),
                e("Q38620", "Cesky Terrier", [], 95),
            ],
        )
    )
    prompts.append(
        prompt(
            "herding-dog",
            "Dog breeds",
            "A herding dog breed",
            [
                e("Q38604", "German Shepherd", ["Alsatian"], 5),
                e("Q38163", "Border Collie", [], 10),
                e("Q38888", "Australian Shepherd", ["Aussie"], 18),
                e("Q38194", "Rough Collie", ["Collie"], 25),
                e("Q38920", "Pembroke Welsh Corgi", ["Corgi"], 28),
                e("Q38220", "Shetland Sheepdog", ["Sheltie"], 35),
                e("Q39120", "Belgian Malinois", ["Malinois"], 45),
                e("Q39200", "Bernese Mountain Dog", [], 48),
                e("Q39010", "Old English Sheepdog", [], 55),
                e("Q39210", "Bearded Collie", [], 70),
                e("Q39310", "Puli", [], 88),
            ],
        )
    )
    prompts.append(
        prompt(
            "toy-dog",
            "Dog breeds",
            "A toy dog breed",
            [
                e("Q37990", "Chihuahua", [], 5),
                e("Q38020", "Pomeranian", [], 12),
                e("Q38180", "Yorkshire Terrier", ["Yorkie"], 15),
                e("Q38210", "Pug", [], 18),
                e("Q38250", "Maltese", [], 30),
                e("Q38300", "Shih Tzu", [], 35),
                e("Q38350", "Papillon", [], 55),
                e("Q38400", "Japanese Chin", [], 72),
                e("Q38450", "English Toy Spaniel", ["King Charles Spaniel"], 80),
                e("Q38500", "Affenpinscher", [], 92),
            ],
        )
    )
    prompts.append(
        prompt(
            "working-dog",
            "Dog breeds",
            "A working dog breed",
            [
                e("Q38800", "Rottweiler", [], 8),
                e("Q38850", "Dobermann", ["Doberman", "Doberman Pinscher"], 12),
                e("Q38900", "Boxer", [], 15),
                e("Q39000", "Siberian Husky", ["Husky"], 18),
                e("Q38950", "Great Dane", [], 22),
                e("Q39100", "Saint Bernard", ["St. Bernard"], 35),
                e("Q39050", "Alaskan Malamute", ["Malamute"], 40),
                e("Q39200", "Bernese Mountain Dog", [], 48),
                e("Q39150", "Newfoundland", [], 55),
                e("Q39250", "Komondor", [], 90),
            ],
        )
    )

    # ——— Chemical elements ———
    prompts.append(
        prompt(
            "chemical-element",
            "Chemical elements",
            "A chemical element",
            [
                e("Q556", "Hydrogen", ["H"], 2),
                e("Q629", "Oxygen", ["O"], 5),
                e("Q623", "Carbon", ["C"], 8),
                e("Q677", "Iron", ["Fe"], 12),
                e("Q897", "Gold", ["Au"], 15),
                e("Q1098", "Uranium", ["U"], 35),
                e("Q1122", "Technetium", ["Tc"], 72),
                e("Q1128", "Promethium", ["Pm"], 85),
                e("Q1307", "Oganesson", ["Og"], 95),
                e("Q1275", "Tennessine", ["Ts"], 98),
            ],
        )
    )
    prompts.append(
        prompt(
            "noble-gas",
            "Chemical elements",
            "A noble gas",
            [
                e("Q560", "Helium", ["He"], 0),
                e("Q654", "Neon", ["Ne"], 20),
                e("Q696", "Argon", ["Ar"], 35),
                e("Q888", "Krypton", ["Kr"], 55),
                e("Q1106", "Xenon", ["Xe"], 70),
                e("Q1133", "Radon", ["Rn"], 85),
                e("Q1307", "Oganesson", ["Og"], 100),
            ],
        )
    )
    prompts.append(
        prompt(
            "lanthanide",
            "Chemical elements",
            "A lanthanide",
            [
                e("Q1385", "Lanthanum", ["La"], 10),
                e("Q1386", "Cerium", ["Ce"], 18),
                e("Q1388", "Neodymium", ["Nd"], 25),
                e("Q1391", "Europium", ["Eu"], 40),
                e("Q1396", "Gadolinium", ["Gd"], 48),
                e("Q1403", "Terbium", ["Tb"], 60),
                e("Q1407", "Dysprosium", ["Dy"], 65),
                e("Q1412", "Holmium", ["Ho"], 75),
                e("Q1420", "Lutetium", ["Lu"], 85),
                e("Q1418", "Thulium", ["Tm"], 90),
            ],
        )
    )
    prompts.append(
        prompt(
            "halogen",
            "Chemical elements",
            "A halogen",
            [
                e("Q688", "Chlorine", ["Cl"], 5),
                e("Q650", "Fluorine", ["F"], 15),
                e("Q1107", "Iodine", ["I"], 25),
                e("Q740", "Bromine", ["Br"], 30),
                e("Q1134", "Astatine", ["At"], 85),
                e("Q1306", "Tennessine", ["Ts"], 100),
            ],
        )
    )
    prompts.append(
        prompt(
            "alkali-metal",
            "Chemical elements",
            "An alkali metal",
            [
                e("Q658", "Sodium", ["Na"], 5),
                e("Q703", "Potassium", ["K"], 8),
                e("Q568", "Lithium", ["Li"], 10),
                e("Q556", "Hydrogen", ["H"], 20),
                e("Q1108", "Caesium", ["Cesium", "Cs"], 45),
                e("Q753", "Rubidium", ["Rb"], 55),
                e("Q1126", "Francium", ["Fr"], 95),
            ],
        )
    )

    # ——— Planets and moons ———
    prompts.append(
        prompt(
            "solar-planet",
            "Planets and moons",
            "A planet in the Solar System",
            [
                e("Q2", "Earth", [], 0),
                e("Q111", "Mars", [], 14),
                e("Q319", "Jupiter", [], 28),
                e("Q313", "Mercury", [], 42),
                e("Q308", "Venus", [], 50),
                e("Q193", "Saturn", [], 57),
                e("Q324", "Uranus", [], 78),
                e("Q332", "Neptune", [], 85),
            ],
        )
    )
    prompts.append(
        prompt(
            "solar-body",
            "Planets and moons",
            "A planet or moon in the Solar System",
            [
                e("Q2", "Earth", [], 0),
                e("Q405", "Moon", ["Luna"], 5),
                e("Q111", "Mars", [], 10),
                e("Q193", "Saturn", [], 25),
                e("Q7547", "Europa", [], 35),
                e("Q15034", "Titan", [], 40),
                e("Q31691", "Ganymede", [], 45),
                e("Q17754", "Enceladus", [], 70),
                e("Q15037", "Iapetus", [], 85),
                e("Q19440", "Hyperion", [], 95),
            ],
        )
    )
    prompts.append(
        prompt(
            "saturn-moon",
            "Planets and moons",
            "A moon of Saturn",
            [
                e("Q15034", "Titan", [], 5),
                e("Q17754", "Enceladus", [], 20),
                e("Q15037", "Iapetus", [], 40),
                e("Q15045", "Rhea", [], 50),
                e("Q15040", "Dione", [], 55),
                e("Q15050", "Tethys", [], 60),
                e("Q17766", "Mimas", [], 70),
                e("Q19440", "Hyperion", [], 75),
                e("Q17958", "Phoebe", [], 85),
                e("Q17758", "Janus", [], 95),
            ],
        )
    )
    prompts.append(
        prompt(
            "jupiter-moon",
            "Planets and moons",
            "A moon of Jupiter",
            [
                e("Q31691", "Ganymede", [], 5),
                e("Q7547", "Europa", [], 10),
                e("Q3123", "Io", [], 15),
                e("Q31693", "Callisto", [], 20),
                e("Q16060", "Amalthea", [], 60),
                e("Q16061", "Himalia", [], 70),
                e("Q16062", "Elara", [], 80),
                e("Q16063", "Pasiphae", [], 88),
                e("Q16064", "Sinope", [], 92),
                e("Q16065", "Lysithea", [], 95),
            ],
        )
    )
    prompts.append(
        prompt(
            "dwarf-planet",
            "Planets and moons",
            "A dwarf planet",
            [
                e("Q339", "Pluto", [], 5),
                e("Q596", "Ceres", [], 25),
                e("Q14928", "Eris", [], 40),
                e("Q10757", "Haumea", [], 60),
                e("Q10756", "Makemake", [], 70),
                e("Q133811", "Gonggong", [], 85),
                e("Q10755", "Quaoar", [], 90),
                e("Q10754", "Sedna", [], 95),
            ],
        )
    )

    # ——— Programming languages ———
    prompts.append(
        prompt(
            "programming-language",
            "Programming languages",
            "A programming language",
            [
                e("Q28865", "Python", [], 2),
                e("Q2005", "JavaScript", ["JS", "ECMAScript"], 5),
                e("Q251", "Java", [], 8),
                e("Q15777", "C", [], 12),
                e("Q2407", "C++", ["Cpp"], 15),
                e("Q37227", "Go", ["Golang"], 25),
                e("Q161053", "Rust", [], 30),
                e("Q2370", "Haskell", [], 55),
                e("Q132874", "Erlang", [], 75),
                e("Q849146", "OCaml", [], 88),
            ],
        )
    )
    prompts.append(
        prompt(
            "compiles-to-js",
            "Programming languages",
            "A language that compiles to JavaScript",
            [
                e("Q349387", "TypeScript", ["TS"], 5),
                e("Q2005", "JavaScript", ["JS"], 10),
                e("Q3770685", "Dart", [], 20),
                e("Q1321919", "CoffeeScript", [], 25),
                e("Q188822", "Elm", [], 55),
                e("Q51798", "ClojureScript", [], 60),
                e("Q4035747", "PureScript", [], 70),
                e("Q15241004", "Reason", ["ReasonML"], 75),
                e("Q726239", "Scala.js", [], 80),
                e("Q105599023", "ReScript", [], 85),
            ],
        )
    )
    prompts.append(
        prompt(
            "systems-language",
            "Programming languages",
            "A systems programming language",
            [
                e("Q15777", "C", [], 5),
                e("Q2407", "C++", [], 10),
                e("Q161053", "Rust", [], 15),
                e("Q37227", "Go", ["Golang"], 20),
                e("Q337562", "Fortran", [], 45),
                e("Q163233", "Zig", [], 55),
                e("Q188822", "Nim", [], 65),
                e("Q5112206", "D", [], 70),
                e("Q426829", "Ada", [], 75),
                e("Q829355", "Oberon", [], 95),
            ],
        )
    )
    prompts.append(
        prompt(
            "scripting-language",
            "Programming languages",
            "A scripting language",
            [
                e("Q28865", "Python", [], 5),
                e("Q2005", "JavaScript", ["JS"], 8),
                e("Q59", "PHP", [], 12),
                e("Q42478", "Ruby", [], 20),
                e("Q131914", "Bash", [], 25),
                e("Q209067", "Lua", [], 35),
                e("Q42470", "Perl", [], 45),
                e("Q51798", "PowerShell", [], 50),
                e("Q46440", "Tcl", [], 75),
                e("Q1201807", "Racket", [], 85),
            ],
        )
    )
    prompts.append(
        prompt(
            "functional-language",
            "Programming languages",
            "A functional programming language",
            [
                e("Q2370", "Haskell", [], 15),
                e("Q726239", "Scala", [], 20),
                e("Q51798", "Clojure", [], 25),
                e("Q163233", "Elixir", [], 30),
                e("Q132874", "Erlang", [], 35),
                e("Q188822", "Elm", [], 40),
                e("Q849146", "OCaml", [], 45),
                e("Q1132275", "F#", ["F Sharp"], 50),
                e("Q34010", "Scheme", [], 55),
                e("Q1201807", "Racket", [], 60),
            ],
        )
    )

    # ——— Linux ———
    prompts.append(
        prompt(
            "linux-distro",
            "Linux distributions",
            "A Linux distribution",
            [
                e("Q215518", "Ubuntu", [], 2),
                e("Q7715973", "Debian", [], 10),
                e("Q58676", "Fedora", [], 15),
                e("Q26876", "Linux Mint", ["Mint"], 18),
                e("Q80737", "Arch Linux", ["Arch"], 20),
                e("Q11241", "openSUSE", [], 35),
                e("Q216794", "Manjaro", [], 40),
                e("Q16471", "Gentoo", [], 55),
                e("Q131382", "Slackware", [], 75),
                e("Q48263", "Void Linux", [], 90),
            ],
        )
    )
    prompts.append(
        prompt(
            "debian-based",
            "Linux distributions",
            "A Debian-based distro",
            [
                e("Q215518", "Ubuntu", [], 2),
                e("Q7715973", "Debian", [], 8),
                e("Q26876", "Linux Mint", ["Mint"], 15),
                e("Q21041353", "Kali Linux", ["Kali"], 25),
                e("Q1295558", "Pop!_OS", ["Pop OS"], 35),
                e("Q15275324", "elementary OS", [], 45),
                e("Q28975100", "Zorin OS", [], 55),
                e("Q21041347", "MX Linux", [], 60),
                e("Q17147124", "Devuan", [], 80),
                e("Q21041344", "AntiX", [], 90),
            ],
        )
    )
    prompts.append(
        prompt(
            "rpm-distro",
            "Linux distributions",
            "An RPM-based Linux distribution",
            [
                e("Q58676", "Fedora", [], 5),
                e("Q28010", "Red Hat Enterprise Linux", ["RHEL"], 10),
                e("Q11241", "openSUSE", [], 20),
                e("Q207783", "CentOS", [], 25),
                e("Q48263", "Rocky Linux", [], 40),
                e("Q16471", "AlmaLinux", [], 45),
                e("Q26876", "Oracle Linux", [], 55),
                e("Q131382", "Mageia", [], 60),
                e("Q80737", "PCLinuxOS", [], 70),
                e("Q215518", "ClearOS", [], 85),
            ],
        )
    )
    prompts.append(
        prompt(
            "arch-based",
            "Linux distributions",
            "An Arch-based Linux distribution",
            [
                e("Q80737", "Arch Linux", ["Arch"], 5),
                e("Q216794", "Manjaro", [], 10),
                e("Q15275324", "EndeavourOS", [], 30),
                e("Q21041353", "Garuda Linux", [], 40),
                e("Q17147124", "Artix Linux", [], 55),
                e("Q25345924", "ArcoLinux", [], 65),
                e("Q28975100", "ArchBang", [], 75),
                e("Q17047990", "Chakra", [], 85),
                e("Q21041347", "Hyperbola", [], 90),
                e("Q28975096", "Parabola", [], 92),
            ],
        )
    )
    prompts.append(
        prompt(
            "mobile-linux",
            "Linux distributions",
            "A mobile Linux distribution",
            [
                e("Q94", "Android", [], 2),
                e("Q152294", "Tizen", [], 35),
                e("Q478762", "Firefox OS", [], 40),
                e("Q16972991", "Ubuntu Touch", [], 45),
                e("Q4042997", "Sailfish OS", [], 50),
                e("Q21041353", "postmarketOS", [], 55),
                e("Q21041347", "Maemo", [], 70),
                e("Q28975100", "Mobian", [], 75),
                e("Q16972790", "Plasma Mobile", [], 65),
                e("Q4042174", "Mer", [], 80),
            ],
        )
    )

    # ——— Tech companies ———
    prompts.append(
        prompt(
            "tech-company",
            "Tech companies",
            "A tech company",
            [
                e("Q312", "Apple", [], 2),
                e("Q2283", "Microsoft", [], 4),
                e("Q95", "Google", [], 6),
                e("Q3884", "Amazon", [], 8),
                e("Q380", "Meta Platforms", ["Meta", "Facebook"], 12),
                e("Q2013", "NVIDIA", ["Nvidia"], 15),
                e("Q37033", "Samsung Electronics", ["Samsung"], 18),
                e("Q7414", "Intel", [], 25),
                e("Q163261", "Sony", [], 30),
                e("Q20800404", "OpenAI", [], 35),
                e("Q1630305", "Palantir", [], 70),
                e("Q855124", "ASML", [], 75),
            ],
        )
    )
    prompts.append(
        prompt(
            "phone-company",
            "Tech companies",
            "A company that makes phones",
            [
                e("Q312", "Apple", [], 2),
                e("Q37033", "Samsung Electronics", ["Samsung"], 5),
                e("Q95", "Google", [], 12),
                e("Q163261", "Sony", [], 20),
                e("Q27978", "Xiaomi", [], 25),
                e("Q2013", "Huawei", [], 30),
                e("Q483551", "Nokia", [], 35),
                e("Q308991", "Motorola", [], 40),
                e("Q11422", "OnePlus", [], 45),
                e("Q855124", "Nothing", [], 80),
            ],
        )
    )
    prompts.append(
        prompt(
            "social-network-company",
            "Tech companies",
            "A social network company",
            [
                e("Q380", "Meta Platforms", ["Meta", "Facebook"], 5),
                e("Q191824", "TikTok", ["ByteDance"], 8),
                e("Q355", "Twitter", ["X"], 10),
                e("Q14001", "LinkedIn", [], 18),
                e("Q179200", "Reddit", [], 20),
                e("Q1630305", "Snap", ["Snapchat"], 25),
                e("Q308962", "Discord", [], 30),
                e("Q246065", "Pinterest", [], 35),
                e("Q8092", "Tumblr", [], 55),
                e("Q4830453", "BeReal", [], 75),
            ],
        )
    )
    prompts.append(
        prompt(
            "chip-company",
            "Tech companies",
            "A semiconductor company",
            [
                e("Q2013", "NVIDIA", ["Nvidia"], 3),
                e("Q7414", "Intel", [], 5),
                e("Q182477", "TSMC", [], 10),
                e("Q1477735", "AMD", [], 12),
                e("Q475080", "Qualcomm", [], 20),
                e("Q11467", "ARM Holdings", ["Arm"], 28),
                e("Q209281", "Broadcom", [], 35),
                e("Q48911", "Texas Instruments", ["TI"], 40),
                e("Q855124", "ASML", [], 45),
                e("Q485046", "Micron Technology", ["Micron"], 50),
            ],
        )
    )
    prompts.append(
        prompt(
            "streaming-company",
            "Tech companies",
            "A streaming / media tech company",
            [
                e("Q907311", "Netflix", [], 3),
                e("Q3884", "Amazon", [], 8),
                e("Q163261", "Disney", [], 10),
                e("Q95", "Google", [], 12),
                e("Q2283", "Microsoft", [], 15),
                e("Q8092", "Spotify", [], 18),
                e("Q246065", "Twitch", [], 25),
                e("Q179200", "Hulu", [], 30),
                e("Q483551", "Roku", [], 55),
                e("Q191824", "Crunchyroll", [], 60),
            ],
        )
    )

    # ——— Pokemon ———
    prompts.append(
        prompt(
            "pokemon-gen1",
            "Pokemon",
            "A Pokémon introduced in Generation I",
            [
                e("Q1998243", "Pikachu", [], 2),
                e("Q12079", "Charizard", [], 5),
                e("Q12085", "Mewtwo", [], 8),
                e("Q12090", "Bulbasaur", [], 15),
                e("Q12095", "Squirtle", [], 18),
                e("Q12100", "Gyarados", [], 25),
                e("Q12105", "Snorlax", [], 30),
                e("Q12110", "Dragonite", [], 35),
                e("Q12120", "Magikarp", [], 40),
                e("Q12150", "Aerodactyl", [], 70),
                e("Q12130", "Kabuto", [], 75),
                e("Q12140", "Omastar", [], 80),
            ],
        )
    )
    prompts.append(
        prompt(
            "water-pokemon",
            "Pokemon",
            "A Water-type Pokémon",
            [
                e("Q12095", "Squirtle", [], 8),
                e("Q12100", "Gyarados", [], 12),
                e("Q12200", "Blastoise", [], 15),
                e("Q12240", "Greninja", [], 18),
                e("Q12120", "Magikarp", [], 20),
                e("Q12230", "Psyduck", [], 22),
                e("Q12210", "Vaporeon", [], 25),
                e("Q12220", "Lapras", [], 30),
                e("Q12260", "Feebas", [], 75),
                e("Q12250", "Wishiwashi", [], 80),
            ],
        )
    )
    prompts.append(
        prompt(
            "fire-pokemon",
            "Pokemon",
            "A Fire-type Pokémon",
            [
                e("Q12079", "Charizard", [], 5),
                e("Q12300", "Charmander", [], 10),
                e("Q12310", "Arcanine", [], 25),
                e("Q12320", "Flareon", [], 30),
                e("Q12330", "Ninetales", [], 35),
                e("Q12350", "Blaziken", [], 38),
                e("Q12340", "Infernape", [], 40),
                e("Q12360", "Cinderace", [], 45),
                e("Q12380", "Torkoal", [], 75),
                e("Q12370", "Numel", [], 80),
            ],
        )
    )
    prompts.append(
        prompt(
            "legendary-pokemon",
            "Pokemon",
            "A legendary Pokémon",
            [
                e("Q12085", "Mewtwo", [], 5),
                e("Q12440", "Rayquaza", [], 20),
                e("Q12410", "Zapdos", [], 22),
                e("Q12400", "Articuno", [], 25),
                e("Q12420", "Moltres", [], 28),
                e("Q12430", "Lugia", [], 30),
                e("Q12450", "Dialga", [], 35),
                e("Q12460", "Giratina", [], 40),
                e("Q12470", "Zygarde", [], 55),
                e("Q12480", "Eternatus", [], 60),
            ],
        )
    )
    prompts.append(
        prompt(
            "starter-pokemon",
            "Pokemon",
            "A starter Pokémon",
            [
                e("Q12300", "Charmander", [], 5),
                e("Q12090", "Bulbasaur", [], 10),
                e("Q12095", "Squirtle", [], 12),
                e("Q12510", "Cyndaquil", [], 38),
                e("Q12500", "Chikorita", [], 40),
                e("Q12520", "Totodile", [], 42),
                e("Q12530", "Treecko", [], 45),
                e("Q12540", "Torchic", [], 48),
                e("Q12550", "Mudkip", [], 50),
                e("Q12560", "Rowlet", [], 55),
            ],
        )
    )

    # ——— Relationship prompts ———
    prompts.append(
        prompt(
            "film-tilda-swinton",
            "Relationship prompts",
            "A film starring Tilda Swinton",
            [
                e("Q190050", "Doctor Strange", [], 5),
                e("Q190110", "The Grand Budapest Hotel", [], 15),
                e(
                    "Q165324",
                    "The Chronicles of Narnia: The Lion, the Witch and the Wardrobe",
                    ["Narnia"],
                    10,
                ),
                e("Q190100", "Moonrise Kingdom", [], 25),
                e("Q190080", "We Need to Talk About Kevin", [], 30),
                e("Q190120", "Suspiria", [], 40),
                e("Q190090", "Only Lovers Left Alive", [], 45),
                e("Q190130", "Michael Clayton", [], 50),
                e("Q190140", "I Am Love", [], 75),
                e("Q190150", "Julia", [], 90),
            ],
        )
    )
    prompts.append(
        prompt(
            "album-radiohead",
            "Relationship prompts",
            "An album by Radiohead",
            [
                e("Q191000", "OK Computer", [], 5),
                e("Q191010", "Kid A", [], 15),
                e("Q191020", "The Bends", [], 20),
                e("Q191030", "In Rainbows", [], 25),
                e("Q191040", "Pablo Honey", [], 40),
                e("Q191050", "Hail to the Thief", [], 45),
                e("Q191060", "A Moon Shaped Pool", [], 50),
                e("Q191070", "Amnesiac", [], 55),
                e("Q191080", "The King of Limbs", [], 70),
                e("Q191090", "I Might Be Wrong", [], 85),
            ],
        )
    )
    prompts.append(
        prompt(
            "car-mazda",
            "Relationship prompts",
            "A car model made by Mazda",
            [
                e("Q192000", "Mazda MX-5", ["Miata", "MX-5"], 5),
                e("Q192010", "Mazda3", ["Mazda 3"], 15),
                e("Q192030", "Mazda CX-5", ["CX-5"], 20),
                e("Q192020", "Mazda6", ["Mazda 6"], 25),
                e("Q192040", "Mazda RX-7", ["RX-7"], 30),
                e("Q192050", "Mazda RX-8", ["RX-8"], 45),
                e("Q192080", "Mazda CX-30", ["CX-30"], 50),
                e("Q192060", "Mazda CX-9", ["CX-9"], 55),
                e("Q192070", "Mazda2", ["Mazda 2", "Demio"], 60),
                e("Q192090", "Mazda Cosmo", [], 90),
            ],
        )
    )
    prompts.append(
        prompt(
            "film-wes-anderson",
            "Relationship prompts",
            "A film directed by Wes Anderson",
            [
                e("Q190110", "The Grand Budapest Hotel", [], 5),
                e("Q190100", "Moonrise Kingdom", [], 15),
                e("Q193000", "The Royal Tenenbaums", [], 20),
                e("Q193010", "Fantastic Mr. Fox", [], 25),
                e(
                    "Q193060",
                    "The Life Aquatic with Steve Zissou",
                    ["The Life Aquatic"],
                    30,
                ),
                e("Q193020", "Isle of Dogs", [], 35),
                e("Q193030", "The French Dispatch", [], 40),
                e("Q193040", "Asteroid City", [], 45),
                e("Q193050", "Rushmore", [], 50),
                e("Q193070", "Bottle Rocket", [], 75),
            ],
        )
    )
    prompts.append(
        prompt(
            "album-beyonce",
            "Relationship prompts",
            "An album by Beyoncé",
            [
                e("Q194000", "Lemonade", [], 5),
                e("Q194020", "Renaissance", [], 10),
                e("Q194010", "Beyoncé", ["Beyonce"], 15),
                e("Q194030", "Cowboy Carter", [], 20),
                e("Q194060", "I Am... Sasha Fierce", ["Sasha Fierce"], 30),
                e("Q194040", "Dangerously in Love", [], 35),
                e("Q194050", "B'Day", ["BDay"], 40),
                e("Q194070", "4", [], 55),
                e("Q194090", "Homecoming: The Live Album", ["Homecoming"], 60),
                e("Q194080", "The Gift", [], 70),
            ],
        )
    )

    cats: dict[str, int] = {}
    for p in prompts:
        cats[p["category"]] = cats.get(p["category"], 0) + 1
        assert len(p["entries"]) >= 6, (p["id"], len(p["entries"]))
        ids = [x["id"] for x in p["entries"]]
        assert len(ids) == len(set(ids)), p["id"]
        labels = [x["label"].lower() for x in p["entries"]]
        assert len(labels) == len(set(labels)), p["id"]

    pack = {
        "pack_id": "core-v1",
        "version": 1,
        "built_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "attribution": ["Wikidata (CC0)", "Wikimedia Pageviews API"],
        "prompts": sorted(prompts, key=lambda p: (p["category"], p["id"])),
    }

    path = "packs/core-v1.json"
    with open(path, "w", encoding="utf-8") as f:
        json.dump(pack, f, indent=2, ensure_ascii=False)
        f.write("\n")

    print("prompts", len(pack["prompts"]))
    print("categories", len(cats), dict(sorted(cats.items())))
    print("min entries", min(len(p["entries"]) for p in pack["prompts"]))
    for pid in ("dog-breed", "country", "solar-planet"):
        p = next(x for x in pack["prompts"] if x["id"] == pid)
        low, high = p["entries"][0], p["entries"][-1]
        print(
            f"spot {pid}: {low['label']} r={low['rarity']}; {high['label']} r={high['rarity']}"
        )
    print("wrote", path)


if __name__ == "__main__":
    main()
