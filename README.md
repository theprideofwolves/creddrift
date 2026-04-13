# CredDrift

CredDrift is an advanced, cross-platform security infrastructure tool designed for the continuous discovery and lifecycle tracking of embedded secrets within source code repositories. Built for Governance, Risk, and Compliance (GRC) workflows, it delivers real-time analytics on secret proliferation, age degradation, and exposure metrics across enterprise workspaces.

## System Architecture

CredDrift leverages a decoupled, local-first architecture. The backend is powered by a high-throughput Go engine interacting with an embedded SQLite datastore, exposing a lightweight API for the UI.

Crucially, the tool implements seamless cross-platform integration via native operating system calls. By utilizing the Go `runtime` package, CredDrift dynamically hooks into native file explorer dialogs (Windows Explorer, macOS Finder, Linux interactive dialogs) to provide unhindered local filesystem access without relying on browser-based directory constraints.

## Architecture Diagram

You can import the following XML directly into diagrams.net (Draw.io) to visualize the system map:

```xml
<mxfile version="21.6.8">
  <diagram name="CredDrift Architecture" id="creddrift-arch-001">
    <mxGraphModel dx="1000" dy="1000" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" pageWidth="850" pageHeight="1100" math="0" shadow="0">
      <root>
        <mxCell id="0" />
        <mxCell id="1" parent="0" />
        <mxCell id="node_frontend" value="Frontend Application&#xa;(HTML5/CSS3/Vanilla JS)" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#1e1e1e;strokeColor=#555555;fontColor=#ffffff;" vertex="1" parent="1">
          <mxGeometry x="320" y="80" width="200" height="60" as="geometry" />
        </mxCell>
        <mxCell id="node_backend" value="Go Backend API &amp; Scanner&#xa;(Native OS Hooks via runtime)" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#2b2b2b;strokeColor=#555555;fontColor=#ffffff;" vertex="1" parent="1">
          <mxGeometry x="320" y="200" width="200" height="60" as="geometry" />
        </mxCell>
        <mxCell id="node_sqlite" value="SQLite Datastore&#xa;(State &amp; GRC Metrics)" style="shape=cylinder3;whiteSpace=wrap;html=1;boundedLbl=1;backgroundOutline=1;size=15;fillColor=#3c3c3c;strokeColor=#777777;fontColor=#ffffff;" vertex="1" parent="1">
          <mxGeometry x="370" y="320" width="100" height="80" as="geometry" />
        </mxCell>
        <mxCell id="edge_1" value="REST / JSON" style="endArrow=classic;html=1;exitX=0.5;exitY=1;exitDx=0;exitDy=0;entryX=0.5;entryY=0;entryDx=0;entryDy=0;strokeColor=#ffffff;" edge="1" parent="1" source="node_frontend" target="node_backend">
          <mxGeometry width="50" height="50" relative="1" as="geometry" />
        </mxCell>
        <mxCell id="edge_2" value="SQL" style="endArrow=classic;html=1;exitX=0.5;exitY=1;exitDx=0;exitDy=0;entryX=0.5;entryY=0;entryDx=0;entryDy=0;entryPerimeter=0;strokeColor=#ffffff;" edge="1" parent="1" source="node_backend" target="node_sqlite">
          <mxGeometry width="50" height="50" relative="1" as="geometry" />
        </mxCell>
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
```

## Secret Detection Logic

CredDrift employs complex pattern matching augmented by information theory to reduce false positives and detect highly randomized cryptographic keys. The scanning engine utilizes Shannon Entropy to quantify the randomness of identified strings.

The entropy $H$ of a given string $X$ is calculated using the following mathematical formula:

$$H(X) = -\sum_{i=1}^{n} P(x_i) \log_2 P(x_i)$$

Where:
*   **$x_i$** represents a discrete character block within the string.
*   **$n$** is the alphabet size (the number of unique characters).
*   **$P(x_i)$** is the probability of the character occurring within the string.

A high entropy score typically indicates a machine-generated cryptographic secret (such as an API key or an RSA private key) rather than standard application text.

## Core GRC Capabilities

*   **Blast Radius Analysis:** Correlates identical secret hashes across disparate file systems to quantify the potential exposure footprint.
*   **Drift Detection:** Tracks the lateral movement of compromised credentials entering new repositories or directories.
*   **Temporal Rotation Tracking:** Enforces security compliance by logging credential lifecycles and validating rotation policies against detected timestamps.

## Technical Installation & Build Process

This tool requires Go 1.21 or higher for optimal performance and dependency resolution.

### 1. Dependency Management

Initialize and verify the necessary Go modules:

```bash
git clone https://github.com/theprideofwolves/creddrift.git
cd creddrift
go mod tidy
go mod verify
```

### 2. Database Initialization

Ensure a clean state for the datastore before executing tests or continuous integration pipelines. Use the following commands to reset the SQLite database:

```bash
# Remove the existing database file
rm -f data/secrets.db

# The backend will automatically rebuild the schema upon execution.
```
*(Note for Windows PowerShell users: use `Remove-Item -Force data\secrets.db`)*

### 3. Optimized Compilation

To deploy CredDrift as a performant, standalone binary with a minimized footprint, compile using optimized linker flags to strip DWARF debugging data and symbol tables:

```bash
# Build binary for the native architecture
go build -ldflags="-s -w" -o build/creddrift main.go
```

## Execution

Initialize the daemon and access the interface:

```bash
./build/creddrift
```

Access the command-and-control dashboard via your browser at `http://localhost:8080`.