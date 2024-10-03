"""
generate_and_check_licenses.py
Utility script for creating the license/LICENSE-3RD-PARTY.md file and reporting if there are any disallowed licenses.
Throws an exception if a disallowed license is found and creates an additional .md file with the disallowed licenses listed.
"""

from io import StringIO
import sys
import os
import pandas as pd

# We can define a subset of licenses we will permit.
ALLOWED_LICENSES = [
    "Apache-2.0",
    "MIT",
    "ISC",
    "BSD-1-Clause",
    "BSD-2-Clause",
    "BSD-3-Clause", # Note that we have not allowed for 4-clause.
    "MPL-2.0",
]

buff = StringIO("")

for l in sys.stdin:
    buff.write(l)

buff.seek(0)

# Read the output into a CSV.
df = pd.read_csv(buff, sep=",", header=None, names=["Module", "License Path", "License"])

# Read the go.mod file to get version and indirect information
go_mod_file = 'go.mod'
with open(go_mod_file, 'r') as f:
    go_mod_content = f.readlines()

# Parse go.mod file to extract versions and indirect status
module_versions = {}
for line in go_mod_content:
    line = line.strip()

    if line:
        parts = line.split()
        if parts[0] == "module" or parts[0] == "go" or parts[0] == ")":
            continue
        if parts[0] == "require":
            if parts[1] == "(":
                continue
            else:
                module_name = parts[1]
                module_version = parts[2]
        else:
            module_name = parts[0]
            module_version = parts[1]

        if "// indirect" in line:
            module_version += " (indirect)"
        module_versions[module_name] = module_version


# Add version and indirect status to the dataframe
df["Module Version"] = df["Module"].map(module_versions)

# Combine module names with versions, excluding 'nan'
df["Module"] = df.apply(
    lambda x: f'{x["Module"]} {x["Module Version"]}' if pd.notna(x["Module Version"]) else x["Module"],
    axis=1
)

# Filter out any Dataphos-repository licenses
df = df[~df["Module"].str.startswith("github.com/dataphos")]

if df.empty:
    print("No third party licenses found.")
    sys.exit()

# Keep only the rows with compliant licenses.
df_allowed = df[df["License"].str.contains("|".join(ALLOWED_LICENSES))]

# Create another dataframe for disallowed licenses.
df_disallowed = df[~df["License"].str.contains("|".join(ALLOWED_LICENSES))]

# Drop the "License Path" and "Module Version" columns from both dataframes
df_allowed = df_allowed.drop(columns=["License Path", "Module Version"])
df_disallowed = df_disallowed.drop(columns=["License Path", "Module Version"])

# Convert the allowed dataframe to markdown
allowed_markdown = df_allowed.to_markdown(index=False)

# Ensure the output directory exists
output_dir = "licenses"
os.makedirs(output_dir, exist_ok=True)

# Write the allowed licenses markdown to a .md file
allowed_output_file = os.path.join(output_dir, "LICENSE-3RD-PARTY.md")
with open(allowed_output_file, 'w') as f:
    f.write(allowed_markdown)

# Display the result.
# If the dataframe isn't empty, the check fails.
if len(df_disallowed.index) > 0:
    disallowed_markdown = df_disallowed.to_markdown(index=False)
    disallowed_output_file = os.path.join(output_dir, "disallowed_licenses.md")
    with open(disallowed_output_file, 'w') as f:
        f.write(disallowed_markdown)
    print(df[["Module", "License"]])
    raise Exception("Found one or more disallowed licenses! Please review dependencies!")

print("No worrisome dependencies found.")
