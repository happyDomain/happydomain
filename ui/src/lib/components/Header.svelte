<script>
 import {
     Navbar,
     NavbarBrand,
     Nav,
     NavItem,
 } from 'sveltestrap';

 import Logo from '$lib/components/Logo.svelte';

 export { className as class };
 let className = '';

 let user_isLogged = false;
</script>

<Navbar
    container={true}
    class="{className} {user_isLogged?'p-0':''}"
>
    <NavbarBrand
        href="/"
        style="padding: 0; margin: -.5rem 0;"
    >
        <Logo />
    </NavbarBrand>

    <Nav navbar>
    </Nav>
    <Nav class="ms-auto text-light" navbar>
        <NavItem>
            asdf
        </NavItem>
    </Nav>
</Navbar>
