from pydantic import BaseModel, Field, field_validator
from typing import List, Dict, Any, Optional, Union
from enum import Enum
import uuid
import datetime


class ProbeName(str, Enum):
    ping = "ping"
    traceroute = "traceroute"
    mtr = "mtr"
    dns = "dns"
    http = "http"
    speedtest = "speedtest"
    bgp = "bgp"
    interface_stats = "interface_stats"
    tcp_stats = "tcp_stats"
    socket_stats = "socket_stats"
    packet_capture = "packet_capture"


class Status(str, Enum):
    OK = "OK"
    WARN = "WARN"
    CRIT = "CRIT"
    NA = "NA"
    # Lowercase variants (Go CLI compatibility)
    ok = "ok"
    warn = "warn"
    fail = "fail"
    na = "na"


class DiagnosisComponent(str, Enum):
    DNS = "DNS"
    Transit = "Transit"
    Peering = "Peering"
    Upstream = "Upstream"
    LocalNetwork = "LocalNetwork"


class Probe(BaseModel):
    name: str  # More flexible - accept any probe name
    status: str  # Accept any status string
    latency_ms: Optional[float] = None
    details: Dict[str, Any] = Field(default_factory=dict)
    error: Optional[str] = None

    @field_validator('status', mode='before')
    @classmethod
    def normalize_status(cls, v):
        if isinstance(v, str):
            # Normalize to uppercase for consistency
            return v.upper() if v.lower() in ['ok', 'warn', 'crit', 'na', 'fail'] else v
        return v


class Diagnosis(BaseModel):
    # Make all fields optional to support Go CLI's simpler format
    component: Optional[str] = None
    confidence: Optional[float] = None
    explanation: Optional[str] = None
    suggested_action: Optional[str] = None
    message: Optional[str] = None  # Go CLI uses this format


class Run(BaseModel):
    run_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    timestamp: datetime.datetime = Field(default_factory=datetime.datetime.utcnow)
    target: str
    mode: str
    score: float
    summary: str

    probes: List[Probe] = Field(default_factory=list)
    diagnosis: List[Diagnosis] = Field(default_factory=list)
    raw: Optional[Dict[str, Any]] = Field(default_factory=dict)

    class Config:
        from_attributes = True  # Updated from orm_mode for Pydantic v2
