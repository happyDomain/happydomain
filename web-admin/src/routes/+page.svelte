<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script lang="ts">
    import {
        Container,
    } from "@sveltestrap/sveltestrap";

    // Example using shared lib
    // import { api } from '$lib/api';

    let stats = {
        totalUsers: 0,
        totalDomains: 0,
        activeProviders: 0
    };

    // TODO: Load stats from admin API
</script>

<Container class="flex-fill my-5">
    <div class="row mb-4">
        <div class="col">
            <h1 class="display-4">
                <i class="bi bi-speedometer2"></i>
                Admin Dashboard
            </h1>
            <p class="text-muted">System overview and management</p>
        </div>
    </div>

    <div class="row row-cols-sm-2 row-cols-lg-3 g-4">
        <div class="col">
            <div class="card">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-center">
                        <div>
                            <h6 class="text-muted mb-1">Total Users</h6>
                            <h2 class="mb-0">{stats.totalUsers}</h2>
                        </div>
                        <div class="text-primary">
                            <i class="bi bi-people-fill" style="font-size: 2rem;"></i>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="col">
            <div class="card">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-center">
                        <div>
                            <h6 class="text-muted mb-1">Total Domains</h6>
                            <h2 class="mb-0">{stats.totalDomains}</h2>
                        </div>
                        <div class="text-primary">
                            <i class="bi bi-globe" style="font-size: 2rem;"></i>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="col">
            <div class="card">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-center">
                        <div>
                            <h6 class="text-muted mb-1">Providers</h6>
                            <h2 class="mb-0">{stats.activeProviders}</h2>
                        </div>
                        <div class="text-primary">
                            <i class="bi bi-hdd-network-fill" style="font-size: 2rem;"></i>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</Container>
