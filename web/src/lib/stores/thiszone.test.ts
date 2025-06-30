import { describe, it, expect, beforeEach, vitest } from 'vitest';
import { get } from 'svelte/store';

import { thisZone, thisAliases, sortedDomains, sortedDomainsWithIntermediate, getZone } from './thiszone';
import type { Zone } from '$lib/model/zone';
import type { Domain } from '$lib/model/domain';

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
        'example.com': [{ _svctype: 'svcs.CNAME', _domain: 'example.com', Service: { Target: 'target.com' } }],
      },
    } as Partial<Zone>;
    thisZone.set(zone as Zone);
    expect(get(thisAliases)).toEqual({ 'target.com': ['example.com'] });
  });

  it('should return sorted domains', () => {
    const zone = {
      services: {
        'b.example.com': [],
        'a.example.com': [],
      },
    } as Partial<Zone>;
    thisZone.set(zone as Zone);
    expect(get(sortedDomains)).toEqual(['a.example.com', 'b.example.com']);
  });

  it('should return sorted domains with intermediate', () => {
    const zone = {
      services: {
        'a.b.example.com': [],
        'b.example.com': [],
      },
    } as Partial<Zone>;
    thisZone.set(zone as Zone);
    const result = get(sortedDomainsWithIntermediate);
    expect(result).toContain('b.example.com');
    expect(result).toContain('a.b.example.com');
  });

  it('should retrieve and set a zone', async () => {
    const domain: Domain = {
      id: 'domain-123',
      id_owner: 'owner-123',
      id_provider: 'provider-123',
      domain: 'example.com',
      group: 'default',
      zone_history: ['123'],
    };
    const zoneId = '123';
    const mockZone = { id: zoneId, name: 'example.com' };

    // Mock de l'API
    globalThis.fetch = vitest.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockZone),
      }) as any
    ) as any;

    const zone = await getZone(domain, zoneId);
    expect(zone).toEqual(mockZone);
    expect(get(thisZone)).toEqual(mockZone);
  });
});
