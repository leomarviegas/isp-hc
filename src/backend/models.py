from pydantic import BaseModel, Field
from typing import List, Dict, Any, Optional
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

class Status(str, Enum):
    OK = "OK"
    WARN = "WARN"
    CRIT = "CRIT"
    NA = "NA"

class DiagnosisComponent(str, Enum):
    DNS = "DNS"
    Transit = "Transit"
    Peering = "Peering"
    Upstream = "Upstream"
    LocalNetwork = "LocalNetwork"

class Probe(BaseModel):
    name: ProbeName
    status: Status
    details: Dict[str, Any] = Field(default_factory=dict)

class Diagnosis(BaseModel):
    component: DiagnosisComponent
    confidence: float
    explanation: str
    suggested_action: str

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
        orm_mode = True
