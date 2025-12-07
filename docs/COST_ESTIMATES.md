# PAW Blockchain - Infrastructure Cost Estimates

**Version:** 1.0
**Last Updated:** 2025-12-07
**Audience:** Finance Teams, Infrastructure Planners, Node Operators

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [AWS Cost Estimates](#aws-cost-estimates)
3. [Google Cloud Platform Cost Estimates](#google-cloud-platform-cost-estimates)
4. [Azure Cost Estimates](#azure-cost-estimates)
5. [Bare Metal / Self-Hosted Estimates](#bare-metal--self-hosted-estimates)
6. [Cost Optimization Strategies](#cost-optimization-strategies)
7. [TCO Analysis (3-Year)](#tco-analysis-3-year)
8. [Cost Breakdown by Component](#cost-breakdown-by-component)

---

## Executive Summary

### Monthly Cost Comparison (Single Validator)

| Provider | Testnet Validator | Mainnet Validator | Mainnet + Monitoring | Notes |
|----------|-------------------|-------------------|----------------------|-------|
| **AWS** | $150 | $350 | $450 | On-demand pricing |
| **AWS (Reserved)** | $95 | $220 | $290 | 1-year commitment, 35% savings |
| **GCP** | $140 | $330 | $420 | On-demand pricing |
| **GCP (Committed)** | $100 | $235 | $305 | 1-year commit, 30% savings |
| **Azure** | $145 | $340 | $435 | Pay-as-you-go |
| **Azure (Reserved)** | $90 | $210 | $275 | 1-year RI, 38% savings |
| **Bare Metal (Colo)** | $250* | $300* | $400* | *Upfront: $3k-5k hardware |

### Annual Cost Comparison (Production Deployment)

**Typical Production Setup:**
- 1 Mainnet Validator
- 2 Sentry Nodes
- 3 API Nodes (load balanced)
- 1 Monitoring Stack

| Provider | Annual Cost (On-Demand) | Annual Cost (Reserved/Committed) | 3-Year TCO |
|----------|-------------------------|----------------------------------|------------|
| **AWS** | $18,500 | $12,000 | $36,000 |
| **GCP** | $17,800 | $12,500 | $37,500 |
| **Azure** | $18,200 | $11,500 | $34,500 |
| **Bare Metal** | $9,600* | $9,600* | $20,800* |

*Bare metal includes $8k upfront hardware + $400/month colo/bandwidth

---

## AWS Cost Estimates

### Instance Types and Pricing

#### Testnet Validator

**Instance:** `c6i.xlarge` (4 vCPU, 8 GB RAM)
```
Compute:        $0.17/hour × 730 hours = $124.10/month
Storage:        200 GB gp3 SSD = $16/month (80 IOPS/GB)
Bandwidth:      100 GB egress = $9/month ($0.09/GB)
Data Transfer:  20 GB ingress = $0 (free)
                ─────────────────────────
Total:          $149/month on-demand
                $95/month (1-year reserved instance, 35% savings)
```

#### Mainnet Validator

**Instance:** `c6i.2xlarge` (8 vCPU, 16 GB RAM) + Enhanced Networking
```
Compute:        $0.34/hour × 730 hours = $248.20/month
Storage:        500 GB gp3 SSD = $40/month
                16,000 IOPS (provisioned) = $30/month
                1,000 MB/s throughput (provisioned) = $20/month
Bandwidth:      500 GB egress = $45/month
EBS Snapshots:  100 GB (backups) = $5/month
                ─────────────────────────
Total:          $388/month on-demand
                $252/month (1-year RI, 35% savings)
                $190/month (3-year RI, 51% savings)
```

**Recommended:** `c6i.2xlarge` with 1-year RI = **$252/month**

#### Alternative: Memory-Optimized Validator

**Instance:** `r6i.2xlarge` (8 vCPU, 64 GB RAM) - for high-throughput networks
```
Compute:        $0.504/hour × 730 hours = $367.92/month
Storage:        500 GB gp3 = $40/month
Bandwidth:      500 GB egress = $45/month
                ─────────────────────────
Total:          $453/month on-demand
                $295/month (1-year RI)
```

#### Sentry Node

**Instance:** `c6i.xlarge` (4 vCPU, 8 GB RAM)
```
Compute:        $124/month
Storage:        200 GB gp3 = $16/month
Bandwidth:      200 GB egress = $18/month
                ─────────────────────────
Total:          $158/month on-demand
                $103/month (1-year RI)
```

#### API Node (Public-Facing)

**Instance:** `c6i.2xlarge` (8 vCPU, 16 GB RAM) behind ALB
```
Compute:        $248/month
Storage:        500 GB gp3 = $40/month
Bandwidth:      2 TB egress = $180/month (high API traffic)
Load Balancer:  Application Load Balancer = $22.50/month
                + $0.008 per LCU-hour (~$50/month)
                ─────────────────────────
Total:          $540/month per API node on-demand
                $440/month (1-year RI)

For 3 API nodes:  $1,320/month (on-demand)
                  $1,200/month (1-year RI)
```

#### Archive Node

**Instance:** `r6i.2xlarge` (8 vCPU, 64 GB RAM)
```
Compute:        $368/month
Storage:        2 TB gp3 = $160/month
                + 6,000 IOPS provisioned = $20/month
S3 Backup:      1 TB (Glacier Deep Archive) = $1/month
Bandwidth:      1 TB egress = $90/month
                ─────────────────────────
Total:          $639/month on-demand
                $420/month (1-year RI)
```

#### Monitoring Stack

**Instance:** `t3.xlarge` (4 vCPU, 16 GB RAM)
```
Compute:        $0.1664/hour × 730 = $121.47/month
Storage:        500 GB gp3 = $40/month
Bandwidth:      50 GB egress = $4.50/month
                ─────────────────────────
Total:          $166/month on-demand
                $108/month (1-year RI)
```

### Complete Production Deployment (AWS)

```
Component                  Quantity    Cost/Month (On-Demand)    Cost/Month (1-Year RI)
─────────────────────────────────────────────────────────────────────────────────────────
Mainnet Validator          1           $388                      $252
Sentry Nodes               2           $316 ($158 × 2)           $206 ($103 × 2)
API Nodes                  3           $1,620 ($540 × 3)         $1,320 ($440 × 3)
Archive Node               1           $639                      $420
Monitoring Stack           1           $166                      $108
─────────────────────────────────────────────────────────────────────────────────────────
Total Monthly:                         $3,129                    $2,306
Total Annual:                          $37,548                   $27,672

Savings with 1-year RI:    26% ($9,876/year)
Savings with 3-year RI:    40% ($15,019/year)
```

### AWS Cost Optimization Tips

1. **Use Reserved Instances (RI):**
   - 1-year standard RI: 30-35% savings
   - 3-year standard RI: 50-55% savings
   - Convertible RI: Flexibility to change instance types (20-30% savings)

2. **Use Spot Instances for Non-Critical Workloads:**
   - Testnet validators: 60-80% savings (acceptable downtime risk)
   - Development environments: Up to 90% savings
   - **Never use Spot for mainnet validators** (uptime critical)

3. **Optimize Storage:**
   - Use `gp3` instead of `gp2` (20% cheaper with better performance)
   - Right-size provisioned IOPS (only provision what you need)
   - Snapshot retention: 7-day retention instead of 30-day (save 75%)

4. **Bandwidth Optimization:**
   - Use CloudFront CDN for static API responses (cache hit = $0.085/GB vs $0.09/GB direct)
   - VPC Peering instead of internet egress (free within region)
   - Use S3 Transfer Acceleration for backups (faster, same cost)

5. **Use Savings Plans:**
   - Compute Savings Plans: 1-year = 30% savings, 3-year = 50% savings
   - More flexible than RIs (apply across instance families)

---

## Google Cloud Platform Cost Estimates

### Instance Types and Pricing

#### Testnet Validator

**Instance:** `n2-standard-4` (4 vCPU, 16 GB RAM)
```
Compute:        $0.194/hour × 730 hours = $141.62/month
Storage:        200 GB SSD Persistent Disk = $34/month
Bandwidth:      100 GB egress = $12/month
                ─────────────────────────
Total:          $187.62/month on-demand
                $130/month (1-year committed use discount, 30%)
```

#### Mainnet Validator

**Instance:** `n2-standard-8` (8 vCPU, 32 GB RAM)
```
Compute:        $0.388/hour × 730 hours = $283.24/month
Storage:        500 GB SSD PD = $85/month
Snapshot:       100 GB snapshots = $2.60/month
Bandwidth:      500 GB egress = $60/month
                ─────────────────────────
Total:          $430.84/month on-demand
                $301/month (1-year CUD, 30%)
                $205/month (3-year CUD, 52%)
```

**Alternative:** `c2-standard-8` (8 vCPU, 32 GB RAM, compute-optimized)
```
Compute:        $0.449/hour × 730 = $327.77/month
Storage:        500 GB SSD = $85/month
Bandwidth:      500 GB = $60/month
                ─────────────────────────
Total:          $472.77/month on-demand
                $331/month (1-year CUD)
```

#### Sentry Node

**Instance:** `n2-standard-4` (4 vCPU, 16 GB RAM)
```
Compute:        $142/month
Storage:        200 GB SSD = $34/month
Bandwidth:      200 GB = $24/month
                ─────────────────────────
Total:          $200/month on-demand
                $140/month (1-year CUD)
```

#### API Node

**Instance:** `n2-standard-8` (8 vCPU, 32 GB RAM) behind Load Balancer
```
Compute:        $283/month
Storage:        500 GB SSD = $85/month
Bandwidth:      2 TB egress = $240/month
Load Balancer:  $18/month + forwarding rules $0.025/hour × 730 = $36.25/month
                ─────────────────────────
Total:          $642.25/month per node on-demand
                $520/month (1-year CUD)

For 3 API nodes:  $1,926.75/month (on-demand)
                  $1,560/month (1-year CUD)
```

#### Monitoring Stack

**Instance:** `n2-standard-4` (4 vCPU, 16 GB RAM)
```
Compute:        $142/month
Storage:        500 GB SSD = $85/month
GCS (logs):     200 GB Standard Storage = $4/month
                ─────────────────────────
Total:          $231/month on-demand
                $162/month (1-year CUD)
```

### Complete Production Deployment (GCP)

```
Component                  Quantity    Cost/Month (On-Demand)    Cost/Month (1-Year CUD)
─────────────────────────────────────────────────────────────────────────────────────────
Mainnet Validator          1           $431                      $301
Sentry Nodes               2           $400 ($200 × 2)           $280 ($140 × 2)
API Nodes                  3           $1,927 ($642 × 3)         $1,560 ($520 × 3)
Archive Node               1           $700                      $490
Monitoring Stack           1           $231                      $162
─────────────────────────────────────────────────────────────────────────────────────────
Total Monthly:                         $3,689                    $2,793
Total Annual:                          $44,268                   $33,516

Savings with 1-year CUD:   24% ($10,752/year)
Savings with 3-year CUD:   40% ($17,707/year)
```

### GCP Cost Optimization Tips

1. **Committed Use Discounts (CUD):**
   - 1-year commitment: 30% savings
   - 3-year commitment: 52% savings
   - Resource-based (vCPU/memory) or spend-based

2. **Preemptible VMs:**
   - 60-91% discount vs on-demand
   - Use for testnets, dev environments, non-critical workloads
   - **Not recommended for validators** (24-hour max runtime)

3. **Sustained Use Discounts:**
   - Automatic discounts for sustained workloads
   - Up to 30% discount for instances running >25% of month
   - **Already applied in on-demand pricing above**

4. **Storage Optimization:**
   - Use Standard Persistent Disk for non-latency-sensitive data (60% cheaper than SSD)
   - Regional Persistent Disk instead of Zonal (same cost, higher availability)
   - Nearline/Coldline Storage for backups ($0.01/GB/month for Coldline)

5. **Network Optimization:**
   - Use Cloud CDN for API responses (cache hit = $0.02/GB vs $0.12/GB egress)
   - Keep traffic within region (free)
   - Use Private Google Access for GCS/BigQuery (no egress charges)

---

## Azure Cost Estimates

### Instance Types and Pricing

#### Testnet Validator

**Instance:** `Standard_F4s_v2` (4 vCPU, 8 GB RAM)
```
Compute:        $0.169/hour × 730 hours = $123.37/month
Storage:        200 GB Premium SSD (P15) = $38.40/month
Bandwidth:      100 GB egress = $8.70/month
                ─────────────────────────
Total:          $170.47/month Pay-as-you-go
                $105/month (1-year Reserved Instance, 38%)
```

#### Mainnet Validator

**Instance:** `Standard_F8s_v2` (8 vCPU, 16 GB RAM)
```
Compute:        $0.338/hour × 730 hours = $246.74/month
Storage:        500 GB Premium SSD (P20) = $76.80/month
Snapshot:       100 GB snapshots = $1.92/month
Bandwidth:      500 GB egress = $43.50/month
                ─────────────────────────
Total:          $368.96/month Pay-as-you-go
                $228/month (1-year RI, 38%)
                $175/month (3-year RI, 53%)
```

**Alternative:** `Standard_D8s_v5` (8 vCPU, 32 GB RAM, balanced)
```
Compute:        $0.385/hour × 730 = $281.05/month
Storage:        500 GB Premium SSD = $76.80/month
Bandwidth:      500 GB = $43.50/month
                ─────────────────────────
Total:          $401.35/month Pay-as-you-go
                $249/month (1-year RI)
```

#### Sentry Node

**Instance:** `Standard_F4s_v2` (4 vCPU, 8 GB RAM)
```
Compute:        $123/month
Storage:        200 GB Premium SSD = $38.40/month
Bandwidth:      200 GB = $17.40/month
                ─────────────────────────
Total:          $178.80/month Pay-as-you-go
                $110/month (1-year RI)
```

#### API Node

**Instance:** `Standard_F8s_v2` (8 vCPU, 16 GB RAM) behind Azure Load Balancer
```
Compute:        $247/month
Storage:        500 GB Premium SSD = $76.80/month
Bandwidth:      2 TB egress = $174/month
Load Balancer:  Standard LB = $21.90/month
                + Data processing $0.005/GB × 2000 GB = $10/month
                ─────────────────────────
Total:          $529.70/month per node Pay-as-you-go
                $420/month (1-year RI)

For 3 API nodes:  $1,589/month (Pay-as-you-go)
                  $1,260/month (1-year RI)
```

#### Monitoring Stack

**Instance:** `Standard_D4s_v5` (4 vCPU, 16 GB RAM)
```
Compute:        $0.192/hour × 730 = $140.16/month
Storage:        500 GB Premium SSD = $76.80/month
Blob Storage:   200 GB Hot tier (logs) = $3.68/month
                ─────────────────────────
Total:          $220.64/month Pay-as-you-go
                $137/month (1-year RI)
```

### Complete Production Deployment (Azure)

```
Component                  Quantity    Cost/Month (PAYG)         Cost/Month (1-Year RI)
─────────────────────────────────────────────────────────────────────────────────────────
Mainnet Validator          1           $369                      $228
Sentry Nodes               2           $358 ($179 × 2)           $220 ($110 × 2)
API Nodes                  3           $1,589 ($530 × 3)         $1,260 ($420 × 3)
Archive Node               1           $650                      $403
Monitoring Stack           1           $221                      $137
─────────────────────────────────────────────────────────────────────────────────────────
Total Monthly:                         $3,187                    $2,248
Total Annual:                          $38,244                   $26,976

Savings with 1-year RI:    30% ($11,268/year)
Savings with 3-year RI:    47% ($17,975/year)
```

### Azure Cost Optimization Tips

1. **Reserved Instances:**
   - 1-year RI: 30-38% savings
   - 3-year RI: 45-53% savings
   - Can exchange for different size (flexibility)

2. **Spot Instances:**
   - Up to 90% discount vs Pay-as-you-go
   - Use for dev/test, non-critical workloads
   - **Not for mainnet validators** (eviction risk)

3. **Azure Hybrid Benefit:**
   - Bring your own Windows Server / SQL Server licenses
   - Up to 49% savings on Windows VMs
   - Not applicable for Linux (PAW uses Linux)

4. **Storage Optimization:**
   - Use Standard SSD (E-series) instead of Premium SSD for non-critical workloads (60% cheaper)
   - Cool/Archive tier for backups ($0.01/GB/month Archive vs $0.0184/GB/month Hot)
   - Managed Disk reservation: Save 12% with 1-year commitment

5. **Network Optimization:**
   - Use Azure CDN for API responses (cache hit = $0.081/GB vs $0.087/GB egress)
   - Keep traffic within region (free)
   - Use VNet peering instead of VPN Gateway ($0.01/GB vs $0.035/GB)

---

## Bare Metal / Self-Hosted Estimates

### Hardware Costs (One-Time)

#### Testnet Validator

**Server:** Dell PowerEdge R450 or HP ProLiant DL325 Gen10 Plus
```
CPU:            AMD EPYC 7313P (16 cores, 3.0 GHz) = $800
RAM:            64 GB DDR4 ECC = $300
Storage:        1 TB NVMe SSD (Samsung PM9A3) = $180
Motherboard:    Included in server chassis = $0
PSU:            Redundant 800W = Included
Chassis:        1U rackmount = Included
NIC:            Dual 10GbE = Included
                ─────────────────────────
Total:          ~$2,500 (complete server)
```

#### Mainnet Validator (High-End)

**Server:** Dell PowerEdge R6525 or Supermicro AS-1114S-WN10RT
```
CPU:            AMD EPYC 7543P (32 cores, 2.8 GHz) = $1,800
RAM:            128 GB DDR4 ECC = $600
Storage:        2 × 2 TB NVMe SSD (RAID 1) = $700
RAID Card:      Hardware RAID controller = $400
PSU:            Redundant 1200W Platinum = Included
Chassis:        2U rackmount = Included
NIC:            Dual 25GbE = $300
                ─────────────────────────
Total:          ~$5,000 (enterprise-grade)
```

### Colocation / Hosting Costs (Monthly)

#### Colocation Facility

**Standard 1U Colocation:**
```
Rack Space:     1U = $50-100/month (varies by location)
Power:          ~200W @ $0.10/kWh = $15/month
Network:        1 Gbps unmetered = $50-150/month
                OR 5 TB/month transfer = $20-50/month
IP Addresses:   1 IPv4 + /64 IPv6 = $5/month
Remote Hands:   $50-100/hour (as needed)
Setup Fee:      $0-200 (one-time)
                ─────────────────────────
Total:          $120-270/month per server

Average:        $200/month per 1U server
```

**Premium Tier 3+ Data Center:**
```
Rack Space:     1U = $150-250/month
Power:          Included (up to 500W)
Network:        10 Gbps unmetered = $200-400/month
DDoS Protection: Included (10-100 Gbps)
Support:        24/7 NOC = Included
                ─────────────────────────
Total:          $350-650/month per server

Average:        $500/month per 1U server (premium)
```

#### Self-Hosted (Home/Office)

**Costs:**
```
Electricity:    200W × 730 hours × $0.12/kWh = $17.52/month
Internet:       1 Gbps fiber = $80-150/month
UPS:            $500 (one-time) / 36 months = $14/month amortized
Cooling:        Minimal (single server) = ~$10/month
                ─────────────────────────
Total:          $121.52/month

Notes:          - Not recommended for mainnet validators (uptime risk)
                - Acceptable for testnets, development
                - Residential IP may have port restrictions
```

### Complete Production Deployment (Bare Metal Colo)

**Upfront Hardware Costs:**
```
1 × Mainnet Validator (high-end)     = $5,000
2 × Sentry Nodes (mid-range)         = $6,000 ($3,000 × 2)
3 × API Nodes (mid-range)            = $9,000 ($3,000 × 3)
1 × Monitoring Server (low-end)      = $2,000
                                       ─────────
Total Hardware:                        $22,000 one-time investment
```

**Monthly Colocation Costs:**
```
Component               Quantity    Colo Cost/Month    Total/Month
──────────────────────────────────────────────────────────────────
Mainnet Validator       1           $250 (premium)     $250
Sentry Nodes            2           $200 (standard)    $400
API Nodes               3           $250 (high BW)     $750
Monitoring              1           $150               $150
──────────────────────────────────────────────────────────────────
Total Monthly:                                         $1,550
Total Annual:                                          $18,600
```

**3-Year TCO (Total Cost of Ownership):**
```
Hardware:               $22,000 (amortized over 3 years = $611/month)
Colocation:             $1,550/month × 36 months = $55,800
Replacement parts:      $2,000 (5-10% of hardware cost over 3 years)
Labor (admin):          $500/month × 36 months = $18,000 (optional)
                        ─────────────────────────
Total 3-Year:           $77,800 (without labor)
                        $95,800 (with 0.25 FTE admin)

Monthly Equivalent:     $2,161/month (without labor)
                        $2,661/month (with labor)

**vs Cloud (AWS 3-year RI):  $2,100/month**
**Break-even analysis: Bare metal is cheaper if:**
  - You already have colo contracts (sunk cost)
  - You operate 5+ servers (economies of scale)
  - You have in-house expertise (no labor cost)
  - Hardware depreciation is 5+ years (longer amortization)
```

---

## Cost Optimization Strategies

### Multi-Cloud Strategy

**Benefits:**
- Avoid vendor lock-in
- Leverage best pricing per service
- Geographic distribution

**Example Hybrid Setup:**
```
Validator 1:        AWS (us-east-1) - $252/month (1-year RI)
Validator 2:        GCP (europe-west4) - $301/month (1-year CUD)
Validator 3:        Azure (westus2) - $228/month (1-year RI)
Sentry Nodes:       AWS + GCP - $412/month total
API Nodes:          CloudFlare Workers + edge caching (pay-per-use)
Monitoring:         Self-hosted (Prometheus Cloud) - $50/month
                    ─────────────────────────
Total:              $1,243/month (58% savings vs single-cloud)
```

**Drawback:** Increased operational complexity

### Right-Sizing

**Common Oversizing Mistakes:**
- Provisioning for peak load instead of average load (use auto-scaling)
- Running same instance type for all workloads (use appropriate types)
- Over-provisioning IOPS (use burstable storage for most workloads)

**Right-Sizing Recommendations:**
```
Scenario: Testnet validator initially provisioned with mainnet specs

Before:     c6i.2xlarge (8 cores, 16 GB) = $248/month
After:      c6i.xlarge (4 cores, 8 GB) = $124/month
Savings:    $124/month (50%)

Scenario: API node with 2 TB bandwidth but only using 200 GB

Before:     CloudFront + EC2 egress = $180/month bandwidth
After:      Right-sized CloudFront caching = $20/month
Savings:    $160/month (89%)
```

### Scheduled Scaling

**For Non-Production Environments:**
```bash
# Automatically stop dev/testnet nodes during off-hours (nights, weekends)
# Example: 12 hours/day, 5 days/week = 60 hours/week = 260 hours/month

Normal cost (730 hours):    $248/month
Scaled cost (260 hours):    $88/month
Savings:                    $160/month (64%)

# AWS Lambda scheduled start/stop
aws lambda create-function --function-name StopDevServers \
  --runtime python3.9 \
  --handler lambda_function.lambda_handler \
  --zip-file fileb://function.zip
```

### Data Transfer Optimization

**Problem:** Egress bandwidth is expensive (especially AWS/Azure)

**Solutions:**
1. **CDN for Static Content:**
   ```
   Direct egress:      1 TB × $0.09/GB = $90/month
   CloudFront:         1 TB × $0.085/GB = $85/month (cache hit)
   Savings:            $5/month (6%) + faster response times
   ```

2. **Compression:**
   ```
   Uncompressed API responses:  1 TB/month
   Gzip compressed (70% ratio): 300 GB/month
   Savings: 700 GB × $0.09 = $63/month (70%)
   ```

3. **Regional Caching:**
   ```
   Deploy API nodes in multiple regions
   Reduce cross-region egress (10× more expensive than same-region)

   Before: All API traffic from us-east-1 → $0.09/GB global egress
   After:  Regional API nodes → $0.01/GB same-region
   Savings: $0.08/GB (89% per GB)
   ```

### Storage Lifecycle Policies

**Automated Storage Tiering:**
```yaml
# AWS S3 Lifecycle Policy for Backups
{
  "Rules": [
    {
      "Id": "MoveOldBackupsToGlacier",
      "Status": "Enabled",
      "Transitions": [
        {
          "Days": 30,
          "StorageClass": "GLACIER"  # $0.004/GB vs $0.023/GB Standard
        },
        {
          "Days": 90,
          "StorageClass": "DEEP_ARCHIVE"  # $0.00099/GB
        }
      ],
      "Expiration": {
        "Days": 365  # Delete after 1 year
      }
    }
  ]
}

# Savings Example:
# 1 TB backups, 365 days retention
Before (S3 Standard):          1000 GB × $0.023 × 12 months = $276/year
After (Glacier Deep Archive):  1000 GB × $0.00099 × 12 months = $11.88/year
Savings:                       $264.12/year (96%)
```

### Discount Programs

1. **AWS:**
   - Savings Plans (better than RIs for flexibility)
   - Enterprise agreements (volume discounts)
   - Activate credits (startups: $1k-100k in credits)

2. **GCP:**
   - Committed Use Discounts
   - Startup programs (Google Cloud for Startups: $200k credits)
   - Non-profit discounts

3. **Azure:**
   - Enterprise Agreement (EA): Custom pricing for large commitments
   - Azure for Startups: $150k credits
   - CSP (Cloud Solution Provider) channel: Negotiated rates

---

## TCO Analysis (3-Year)

### Scenario 1: AWS Reserved Instances (Conservative)

```
Year 1:
  Validator (1-year RI):      $252/month × 12 = $3,024
  Sentries (1-year RI):       $206/month × 12 = $2,472
  APIs (1-year RI):           $1,320/month × 12 = $15,840
  Monitoring (1-year RI):     $108/month × 12 = $1,296
  Total Year 1:               $22,632

Year 2:
  Renew 1-year RIs:           $22,632

Year 3:
  Renew 1-year RIs:           $22,632

3-Year Total:                 $67,896

Alternative: 3-Year RI Upfront:
  Validator (3-year RI):      $190/month × 36 = $6,840
  Sentries:                   $160/month × 36 = $5,760
  APIs:                       $1,100/month × 36 = $39,600
  Monitoring:                 $85/month × 36 = $3,060
  3-Year Total:               $55,260

Savings (3-year vs annual RIs): $12,636 (19%)
```

### Scenario 2: GCP Committed Use (Aggressive)

```
3-Year CUD (52% discount):
  Validator:                  $205/month × 36 = $7,380
  Sentries:                   $196/month × 36 = $7,056
  APIs:                       $1,350/month × 36 = $48,600
  Monitoring:                 $120/month × 36 = $4,320
  3-Year Total:               $67,356
```

### Scenario 3: Bare Metal (Colo)

```
Upfront Hardware:             $22,000
Year 1:
  Colocation:                 $1,550/month × 12 = $18,600
  Replacement parts:          $500
  Total Year 1:               $19,100

Year 2:
  Colocation:                 $18,600
  Replacement parts:          $500
  Total Year 2:               $19,100

Year 3:
  Colocation:                 $18,600
  Replacement parts:          $1,000 (aging hardware)
  Total Year 3:               $19,600

3-Year Total:                 $22,000 + $57,800 = $79,800

Note: Higher than cloud for small deployments
      Becomes cost-effective at 10+ servers
```

### TCO Summary (3-Year)

| Approach | 3-Year Cost | Monthly Equivalent | Best For |
|----------|-------------|-------------------|----------|
| **AWS 3-Year RI** | $55,260 | $1,535 | Stable, predictable workload |
| **AWS 1-Year RI** | $67,896 | $1,886 | Flexibility, uncertain growth |
| **GCP 3-Year CUD** | $67,356 | $1,871 | GCP ecosystem preference |
| **Azure 3-Year RI** | $52,920 | $1,470 | Lowest cost, Azure expertise |
| **Bare Metal Colo** | $79,800 | $2,217 | Control, compliance, 10+ servers |
| **On-Demand Cloud** | $112,644 | $3,129 | Short-term, testing |

**Recommendation:** Azure 3-Year RI for lowest TCO ($52,920)

---

## Cost Breakdown by Component

### Validator Operating Costs (Mainnet, 1-Year)

```
Component               Percentage    Monthly Cost    Annual Cost
────────────────────────────────────────────────────────────────
Compute (EC2/VM)        65%           $164            $1,968
Storage (EBS/SSD)       15%           $38             $456
Network (Bandwidth)     12%           $30             $360
Backup/Snapshots        3%            $8              $96
Monitoring/Logging      3%            $8              $96
Management Overhead     2%            $4              $48
────────────────────────────────────────────────────────────────
Total                   100%          $252            $3,024
```

**Cost Drivers:**
1. **Compute (65%):** Can't reduce (consensus requires performance)
2. **Storage (15%):** Optimize via pruning, tiering
3. **Network (12%):** Optimize via regional deployment, compression

---

**For additional cost optimization guidance:**
- See [RESOURCE_REQUIREMENTS.md](RESOURCE_REQUIREMENTS.md) for right-sizing
- See [PERFORMANCE_TUNING.md](PERFORMANCE_TUNING.md) for efficiency gains
- Contact your cloud account manager for enterprise pricing
