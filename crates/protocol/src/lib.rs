//! Wire types exchanged between the DeployOS control plane and node agents.
//!
//! This crate defines the message shapes only; transport (HTTP, gRPC, QUIC, ...)
//! is decided by the components that depend on it.

use serde::{Deserialize, Serialize};

/// Uniquely identifies a single DeployOS node agent within a fleet.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct AgentId(pub String);

impl AgentId {
    pub fn new(id: impl Into<String>) -> Self {
        Self(id.into())
    }
}

/// A periodic liveness signal sent from an agent to the control plane.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Heartbeat {
    pub agent_id: AgentId,
    pub agent_version: String,
    /// Unix timestamp, in seconds, of when the heartbeat was produced.
    pub timestamp: u64,
}

impl Heartbeat {
    pub fn new(agent_id: AgentId, agent_version: impl Into<String>, timestamp: u64) -> Self {
        Self {
            agent_id,
            agent_version: agent_version.into(),
            timestamp,
        }
    }

    pub fn to_json(&self) -> serde_json::Result<String> {
        serde_json::to_string(self)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn heartbeat_round_trips_through_json() {
        let hb = Heartbeat::new(AgentId::new("node-1"), "0.0.0", 1_700_000_000);
        let json = hb.to_json().expect("serialize");
        let decoded: Heartbeat = serde_json::from_str(&json).expect("deserialize");
        assert_eq!(hb, decoded);
    }
}
