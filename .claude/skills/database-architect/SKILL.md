---
name: database-architect
description: Design, analyze, or refactor database schemas across any paradigm. Trigger this skill when the user mentions databases, schemas, SQL, NoSQL, MongoDB, PostgreSQL, DynamoDB, ERDs, access patterns, normalization, or data modeling for software implementations.
---

# Universal Database Schema Architect

## Goal

Act as a principal data architect. Analyze software implementation details, determine the optimal database paradigm (Relational, Document, Wide-Column, or Search/Vector), and design a highly scalable, optimized schema tailored to the application's specific read/write patterns and business logic.

## Phase 1: Paradigm Selection & Access Analysis

Before writing any schemas, analyze the requirements to determine the correct data store:

1.  **Extract Access Patterns**: Identify the exact read/write queries, read-to-write ratio, and latency requirements.
2.  **Determine Paradigm**:
    * **Relational (SQL)**: Default for structured data with complex, unpredictable queries, strict ACID transaction requirements, and clear entity relationships.
    * **Document/Wide-Column (NoSQL)**: Choose for massive read/write scale, flexible schemas, or when access patterns are predictable and heavily favor read-efficiency.
    * **Search/Vector (Elasticsearch, pgvector)**: Recommend if the requirements involve AI agents, Retrieval-Augmented Generation (RAG), semantic search, or heavy full-text aggregations.



## Phase 2: Schema Design Execution
Follow the specific rules for the chosen paradigm.

### Path A: Relational (SQL) Design
1.  **Entity & Relationship Mapping**: Identify core entities and define cardinality (1:1, 1:N, M:N). 
2.  **Normalization**: Ensure the design achieves at least 3NF to eliminate data redundancy.
3.  **Output Requirements**:
    * Generate a `mermaid` ERD diagram.
    * Provide SQL DDL statements (default to PostgreSQL).
    * Include primary keys (UUID/BIGINT), foreign key constraints (`ON DELETE`), standard timestamps, and essential indexes.

### Path B: NoSQL Design
1.  **The Embed vs. Reference Decision**:
    * *Embed*: For 1:1 or bounded 1:Few relationships accessed together.
    * *Reference*: For M:N relationships or unbounded arrays.
2.  **Key Design**: For systems like DynamoDB, define the Partition Key (PK) and Sort Key (SK) to ensure even data distribution.
3.  **Output Requirements**:
    * Provide JSON structures for Document DBs with inline comments explaining embedding choices.
    * Provide an Access Pattern routing table mapping queries to PKs, SKs, and GSIs for Wide-Column DBs.



## Phase 3: Implementation Assessment & Code Review
If analyzing existing code or user stories, flag specific anti-patterns based on the database type:
* **SQL Risks**: N+1 queries in ORMs, missing foreign key indexes, or lock contention risks in high-concurrency transactions.
* **NoSQL Risks**: Unbounded arrays inside documents, hot partitions, or complex multi-document transactions that will degrade performance.
* **Hybrid Sync**: If the architecture uses a primary DB alongside a search engine or vector store, specify the syncing mechanism (e.g., change data capture, event-driven updates) to maintain eventual consistency.

## Edge Cases & Constraints
* **Soft Deletes**: Suggest `deleted_at` columns if data retention or historical auditing is mentioned.
* **State Management**: If the schema requires tracking status changes (e.g., in a LangChain or agentic workflow), ensure there is a clear mechanism for state persistence and history logging.