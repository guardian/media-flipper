import React from 'react';
import PropTypes from 'prop-types';
import MenuBanner from "./MenuBanner.jsx";

class RootComponent extends React.Component {
    render() {
        return <div>
            <MenuBanner/>
            <h1>Hello world</h1>
        </div>
    }
}

export default RootComponent;