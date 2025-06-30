import { describe, it, expect, beforeEach, vitest } from 'vitest';
import { get } from 'svelte/store';

import { thisZone, thisAliases, sortedDomains, sortedDomainsWithIntermediate, getZone } from './thiszone';

describe('Zone Store', () => {
  beforeEach(() => {
    thisZone.set(null); // Réinitialiser le store avant chaque test
  });

  it('should initialize thisZone as null', () => {
    expect(get(thisZone)).toBeNull();
  });

  it('should compute aliases correctly', () => {
    const zone = {
      services: {
        'example.com': [{ _svctype: 'svcs.CNAME', Service: { Target: 'target.com' } }],
      },
    };
    thisZone.set(zone);
    expect(get(thisAliases)).toEqual({ 'target.com': ['example.com'] });
  });

  it('should return sorted domains', () => {
    const zone = {
      services: {
        'b.example.com': [],
        'a.example.com': [],
      },
    };
    thisZone.set(zone);
    expect(get(sortedDomains)).toEqual(['a.example.com', 'b.example.com']);
  });

  it('should return sorted domains with intermediate', () => {
    const zone = {
      services: {
        'a.b.example.com': [],
        'b.example.com': [],
      },
    };
    thisZone.set(zone);
    const result = get(sortedDomainsWithIntermediate);
    expect(result).toContain('b.example.com');
    expect(result).toContain('a.b.example.com');
  });

  it('should retrieve and set a zone', async () => {
    const domain = { name: 'example.com' };
    const zoneId = '123';
    const mockZone = { id: zoneId, name: 'example.com' };

    // Mock de l'API
    global.fetch = vitest.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockZone),
      })
    );

    const zone = await getZone(domain, zoneId);
    expect(zone).toEqual(mockZone);
    expect(get(thisZone)).toEqual(mockZone);
  });
});
