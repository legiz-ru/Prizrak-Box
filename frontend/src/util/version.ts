export function normalizeVersion(version?: string | null): string {
    if (!version) {
        return '';
    }

    const trimmed = version.trim();
    if (!trimmed) {
        return '';
    }

    const withoutPrefix = trimmed.replace(/^v/i, '');
    const core = withoutPrefix.split(/[+\-]/)[0];
    return core.trim();
}

export function compareVersions(a?: string | null, b?: string | null): number {
    const left = normalizeVersion(a);
    const right = normalizeVersion(b);

    if (!left && !right) {
        return 0;
    }
    if (!left) {
        return -1;
    }
    if (!right) {
        return 1;
    }

    const leftParts = left.split('.').map((segment) => parseInt(segment, 10) || 0);
    const rightParts = right.split('.').map((segment) => parseInt(segment, 10) || 0);
    const length = Math.max(leftParts.length, rightParts.length);

    for (let index = 0; index < length; index += 1) {
        const leftValue = leftParts[index] ?? 0;
        const rightValue = rightParts[index] ?? 0;
        if (leftValue > rightValue) {
            return 1;
        }
        if (leftValue < rightValue) {
            return -1;
        }
    }

    return 0;
}

export function resolveVersionLabel(tag?: string | null, name?: string | null): string {
    if (name && name.trim()) {
        return name.trim();
    }

    if (tag && tag.trim()) {
        return tag.trim();
    }

    return '';
}
