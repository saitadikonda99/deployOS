use std::time::{SystemTime, UNIX_EPOCH};

use deployos_common::version::VERSION;
use deployos_protocol::{AgentId, Heartbeat};

fn agent_id() -> AgentId {
    let id =
        std::env::var("DEPLOYOS_AGENT_ID").unwrap_or_else(|_| "unregistered-agent".to_string());
    AgentId::new(id)
}

fn unix_timestamp() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("system clock is before the Unix epoch")
        .as_secs()
}

fn main() {
    let heartbeat = Heartbeat::new(agent_id(), VERSION, unix_timestamp());
    let json = heartbeat
        .to_json()
        .expect("heartbeat is always serializable");
    println!("{json}");
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn unix_timestamp_is_after_2020() {
        assert!(unix_timestamp() > 1_577_836_800);
    }
}
