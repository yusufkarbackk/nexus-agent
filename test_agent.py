"""
Nexus Agent Test Script
Tests sending data through the local Nexus Agent
The agent handles encryption and forwarding to Nexus API

Usage:
    python test_agent.py

Make sure the nexus-agent is running locally first!
"""

import requests
import json
from datetime import datetime

# ============================================
# CONFIGURATION
# ============================================
AGENT_URL = "http://localhost:9000"  # Local nexus-agent endpoint
APP_KEY = "app_wvvKeZcwYeT2xDA8"      # Your app key from Nexus UI
MASTER_SECRET = "cLDSw9+DtOXPjNplZIhlGIgO55PBR6KhM67tni9y54w="
# ============================================


def test_send_data():
    """Send test data through the nexus-agent"""
    
    print("=" * 50)
    print("Nexus Agent Test")
    print("=" * 50)
    
    # Test data to send
    test_data = {
        "title_2": "Hello World",
        "body_2": "Hello body",
        "userId": 1,
    }
    
    print(f"\n📡 Agent URL: {AGENT_URL}")
    print(f"🔑 App Key: {APP_KEY}")
    print(f"\n📤 Sending test data:")
    print(f"   {json.dumps(test_data, indent=2)}")
    
    # Send to agent's /send endpoint
    # The agent expects: { "app_key": "...", "data": {...} }
    payload = {
        "app_key": APP_KEY,
        "data": test_data,
    }
    
    try:
        response = requests.post(
            f"{AGENT_URL}/send",
            json=payload,
            headers={
                "Content-Type": "application/json"
            },
            timeout=10
        )
        
        print(f"\n📥 Response Status: {response.status_code}")
        
        try:
            response_data = response.json()
            print(f"📥 Response Body:")
            print(f"   {json.dumps(response_data, indent=2)}")
        except:
            print(f"📥 Response Body: {response.text}")
        
        if response.status_code == 200:
            print("\n✅ SUCCESS! Data was sent through the agent!")
            print("\nThe agent should have:")
            print("  1. Received your unencrypted data")
            print("  2. Encrypted it with AES-256-GCM")
            print("  3. Forwarded to Nexus API")
            print("\nCheck the Logs page in Nexus UI to verify!")
        else:
            print(f"\n⚠️ Agent returned status {response.status_code}")
            
    except requests.exceptions.ConnectionError:
        print("\n❌ Connection Error!")
        print("\nIs the nexus-agent running?")
        print("  cd nexus-agent")
        print("  go run ./cmd/agent")
        print("\nOr check if it's running on a different port.")
        
    except Exception as e:
        print(f"\n❌ Error: {e}")


def test_health():
    """Check if agent is healthy"""
    
    print("\n🩺 Checking agent health...")
    
    try:
        response = requests.get(f"{AGENT_URL}/health", timeout=5)
        
        if response.status_code == 200:
            print("✅ Agent is healthy!")
            try:
                health_data = response.json()
                print(f"   {json.dumps(health_data, indent=2)}")
            except:
                print(f"   {response.text}")
            return True
        else:
            print(f"⚠️ Agent returned status {response.status_code}")
            return False
            
    except requests.exceptions.ConnectionError:
        print("❌ Cannot connect to agent!")
        return False


def main():
    print("\n" + "=" * 50)
    print("  NEXUS AGENT TEST SCRIPT")
    print("=" * 50)
    
    # First check health
    if test_health():
        print()
        test_send_data()
    else:
        print("\n❌ Agent is not running or not reachable")
        print("\nStart the agent with:")
        print("  cd nexus-agent")
        print("  go run ./cmd/agent")


if __name__ == "__main__":
    main()
