# Groups Page Specification

## Overview
The Groups page provides a powerful yet intuitive interface for managing Ansible inventory groups using collapsible sections with host cards. This design maintains consistency with the existing stacks view while providing hierarchical group management capabilities.

## UI Design: Collapsible Sections with Host Cards

### Visual Hierarchy
```
Groups Page
├── Filter Bar (tags, status, search)
├── Quick Actions (Add Group, Bulk Operations)
└── Group Sections (collapsible)
    ├── Parent Group Header
    │   ├── Group Info (name, host count, tags, description)
    │   ├── Group Actions (edit, delete, add host)
    │   └── Expansion Controls
    ├── Subgroups (nested, same structure)
    └── Host Cards Grid
        ├── Host Card (name, IP, tags, services status)
        └── Host Actions (edit, remove from group)
```

### Key Components

#### Group Section Header
```tsx
interface GroupHeaderProps {
  group: Group;
  hostCount: number;
  isExpanded: boolean;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onAddHost: () => void;
}
```

Visual Design:
- **Left**: Expand/collapse chevron + group name + host count badge
- **Center**: Tags chips + description text (truncated)
- **Right**: Action buttons (Edit, Delete, Add Host)
- **Background**: Same card styling as stacks view
- **States**: Collapsed (summary only), Expanded (shows content)

#### Host Card (within groups)
```tsx
interface GroupHostCardProps {
  host: Host;
  group: Group;
  onEdit: () => void;
  onRemoveFromGroup: () => void;
  serviceStatus?: ServiceStatus;
}
```

Visual Design:
- **Compact version** of existing host cards from stacks view
- **Top row**: Host name + IP address
- **Middle row**: Tags as small chips
- **Bottom row**: Service status dots + action buttons
- **Actions**: Edit host, Remove from group
- **Drag handle**: For reordering within group

### Layout Patterns

#### 1. Collapsed Group
```
┌─ Infrastructure (12 hosts) ──────────────── ▼ [Edit] [+Host] [Delete]
│ Tags: critical, production, monitoring
│ Description: Core infrastructure services and networking
└─────────────────────────────────────────────────────────────────────
```

#### 2. Expanded Group with Subgroups
```
┌─ Infrastructure (12 hosts) ──────────────── ▲ [Edit] [+Host] [Delete]
│ Tags: critical, production, monitoring
│ Description: Core infrastructure services and networking
│ 
│ ┌─ DNS (2 hosts) ────────────── ▼ [Edit] [+Host] [Delete]
│ │ Tags: dns, primary, secondary
│ │ Description: DNS resolution services
│ └─────────────────────────────────────────────────────────────
│
│ ┌─ Docker (8 hosts) ───────────── ▲ [Edit] [+Host] [Delete]
│ │ Tags: docker, containerization
│ │ Description: Container platform hosts
│ │
│ │ ┌─ anchorage ──────────┐ ┌─ dock ─────────────┐
│ │ │ 10.30.1.122         │ │ 10.55.6.136       │
│ │ │ gpu, ml, production │ │ gpu, dev, testing │
│ │ │ ●●●●● (12 services) │ │ ●●○○○ (8 services)│
│ │ │ [Edit] [Remove]     │ │ [Edit] [Remove]   │
│ │ └─────────────────────┘ └───────────────────┘
│ │
│ │ ┌─ harbormaster ──────┐ ┌─ lighthouse ──────┐
│ │ │ 10.24.7.55          │ │ 10.55.6.137       │
│ │ │ management, infra   │ │ monitoring, obs   │
│ │ │ ●●●○○ (6 services)  │ │ ●●●●○ (9 services)│
│ │ │ [Edit] [Remove]     │ │ [Edit] [Remove]   │
│ │ └─────────────────────┘ └───────────────────┘
│ └─────────────────────────────────────────────────────────────
└─────────────────────────────────────────────────────────────────────
```

## Data Structures

### Group Interface
```typescript
interface Group {
  id: string;
  name: string;
  description?: string;
  tags: string[];
  parentId?: string;
  children?: Group[];
  hosts: Host[];
  vars?: Record<string, string>;
  owner: string;
  createdAt: Date;
  updatedAt: Date;
}
```

### Enhanced Host Interface
```typescript
interface Host {
  // Existing fields
  id: string;
  name: string;
  addr: string;
  vars: Record<string, string>;
  groups: string[];
  labels: Record<string, string>;
  owner: string;
  
  // New fields for group management
  tags: string[];
  description?: string;
  groupMemberships: GroupMembership[];
}

interface GroupMembership {
  groupId: string;
  groupName: string;
  directMember: boolean; // true if directly assigned, false if inherited
  inheritedFrom?: string; // parent group ID if inherited
}
```

## Database Schema Extensions

### Groups Table
```sql
CREATE TABLE groups (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    tags TEXT[] DEFAULT '{}',
    parent_id BIGINT REFERENCES groups(id) ON DELETE CASCADE,
    vars JSONB DEFAULT '{}',
    owner TEXT NOT NULL DEFAULT 'unassigned',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_groups_parent_id ON groups(parent_id);
CREATE INDEX idx_groups_tags ON groups USING GIN(tags);
CREATE INDEX idx_groups_owner ON groups(owner);
```

### Host-Group Relationships
```sql
CREATE TABLE host_groups (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    direct_member BOOLEAN DEFAULT TRUE,
    inherited_from BIGINT REFERENCES groups(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(host_id, group_id)
);

CREATE INDEX idx_host_groups_host_id ON host_groups(host_id);
CREATE INDEX idx_host_groups_group_id ON host_groups(group_id);
```

### Enhanced Hosts Table
```sql
-- Add tags column to existing hosts table
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';
ALTER TABLE hosts ADD COLUMN IF NOT EXISTS description TEXT;

CREATE INDEX IF NOT EXISTS idx_hosts_tags ON hosts USING GIN(tags);
```

## API Endpoints

### Group Management
```typescript
// GET /api/groups - List all groups with hierarchy
interface GroupsResponse {
  groups: Group[];
  total: number;
}

// GET /api/groups/:id - Get specific group with hosts
interface GroupDetailResponse {
  group: Group;
  hosts: Host[];
  subgroups: Group[];
}

// POST /api/groups - Create new group
interface CreateGroupRequest {
  name: string;
  description?: string;
  tags: string[];
  parentId?: string;
  vars?: Record<string, string>;
}

// PUT /api/groups/:id - Update group
interface UpdateGroupRequest {
  name?: string;
  description?: string;
  tags?: string[];
  parentId?: string;
  vars?: Record<string, string>;
}

// DELETE /api/groups/:id - Delete group (handles cascading)

// POST /api/groups/:id/hosts - Add hosts to group
interface AddHostsToGroupRequest {
  hostIds: string[];
}

// DELETE /api/groups/:id/hosts/:hostId - Remove host from group
```

### Host Management (Enhanced)
```typescript
// PUT /api/hosts/:id - Update host with tags and description
interface UpdateHostRequest {
  // Existing fields...
  tags?: string[];
  description?: string;
}

// GET /api/hosts/:id/groups - Get host's group memberships
interface HostGroupsResponse {
  memberships: GroupMembership[];
}
```

## Enhanced Inventory Processing

### Inventory Format Support
The system will parse the enhanced inventory format:

```yaml
all:
  hosts:
    # tags: automation,management,controller
    ant-parade:
      ansible_host: 10.30.1.111
      ansible_user: kai
      # description: Primary automation controller
    
    # tags: docker,gpu,ml,production
    anchorage:
      ansible_host: 10.30.1.122
      # description: High-performance GPU server for ML/AI workloads

# tags: hypervisor,infrastructure,critical
proxmox:
  # description: Proxmox virtualization cluster
  hosts:
    avocado:
    bamboo:
  vars:
    ansible_user: root

# tags: automation,configuration,infrastructure  
ansible:
  # description: Automation and configuration management nodes
  hosts:
    ant-parade:
    leaf-cutter:
```

### Parsing Logic Enhancements
```go
// Enhanced Host struct
type Host struct {
    Name        string            `json:"name"`
    Addr        string            `json:"addr"`
    Vars        map[string]string `json:"vars,omitempty"`
    Groups      []string          `json:"groups,omitempty"`
    Owner       string            `json:"owner,omitempty"`
    Tags        []string          `json:"tags,omitempty"`        // NEW
    Description string            `json:"description,omitempty"` // NEW
}

// Enhanced Group struct
type Group struct {
    Name        string            `json:"name"`
    Description string            `json:"description,omitempty"`
    Tags        []string          `json:"tags,omitempty"`
    Hosts       []string          `json:"hosts,omitempty"`
    Children    []string          `json:"children,omitempty"`
    Vars        map[string]string `json:"vars,omitempty"`
}
```

## User Experience Features

### 1. Smart Defaults
- **Most-used groups** expanded on page load
- **Recently modified** groups highlighted
- **Empty groups** shown collapsed with visual indicator

### 2. Filtering & Search
- **Tag-based filtering**: Same chip-based system as stacks view
- **Text search**: Searches group names, descriptions, and host names
- **Status filtering**: Show groups by host health status
- **Owner filtering**: Filter by group/host owner

### 3. Drag & Drop Operations
- **Reorder hosts** within groups
- **Move hosts** between groups
- **Reorder groups** within parent groups
- **Visual feedback** during drag operations

### 4. Bulk Operations
- **Multi-select hosts** across different groups
- **Bulk tag editing** for selected hosts
- **Mass group assignment** for selected hosts
- **Bulk host operations** (restart services, run commands)

### 5. Keyboard Navigation
- **Arrow keys** to navigate between groups/hosts
- **Enter/Space** to expand/collapse groups
- **Tab navigation** through action buttons
- **Escape** to clear selections

## Implementation Priority

### Phase 1: Foundation
1. Database schema migrations
2. Enhanced inventory parsing
3. Basic API endpoints (CRUD operations)
4. Group data structures

### Phase 2: Core UI
1. Group section component
2. Host card component (group variant)
3. Basic expand/collapse functionality
4. Group creation/editing forms

### Phase 3: Advanced Features
1. Drag & drop operations
2. Bulk selection and operations
3. Filtering and search
4. Keyboard navigation

### Phase 4: Polish
1. Animation and transitions
2. Loading states and error handling
3. Accessibility improvements
4. Performance optimizations

## Success Metrics

### Usability
- **Quick group overview** without expanding (host count, tags visible)
- **Efficient host management** (drag & drop, bulk operations)
- **Clear hierarchy** visualization without confusion
- **Fast navigation** between groups and hosts

### Consistency
- **Visual parity** with stacks view (colors, buttons, cards)
- **Interaction patterns** match existing UI conventions
- **Loading states** follow same skeleton UI approach
- **Error handling** uses same notification system

### Performance
- **Lazy loading** for large group hierarchies
- **Virtualized rendering** for groups with many hosts
- **Smart caching** of group/host relationships
- **Efficient updates** without full page reloads

This specification provides a comprehensive foundation for implementing the Groups page while maintaining consistency with the existing DD-UI interface and ensuring powerful yet intuitive group management capabilities.