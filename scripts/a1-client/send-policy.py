# a1_client.py
#
# SPDX-FileCopyrightText: 2022-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

import requests
import json
import sys

def main():
    if len(sys.argv) != 2:
        print("Usage: python a1_client.py <policy_file>")
        sys.exit(1)

    policy_file = sys.argv[1]
    with open(policy_file, 'r') as f:
        policy = json.load(f)

    a1_mediator_url = "http://localhost:10000" # Assuming a port-forward to the a1mediator service
    policy_id = policy.get("policy_id")
    url = f"{a1_mediator_url}/a1-p/policytypes/{policy.get('policy_type_id')}/policies/{policy_id}"

    print(f"Sending policy to {url}")
    try:
        response = requests.put(url, json=policy, timeout=5)
        response.raise_for_status()
        print(f"Policy {policy_id} created successfully")
    except requests.exceptions.RequestException as e:
        print(f"Error creating policy: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
