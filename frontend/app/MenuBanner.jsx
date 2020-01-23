import React from 'react';
import PropTypes from 'prop-types';
import css from './MenuBanner.css';
import {Link} from 'react-router-dom';

class MenuBanner extends React.Component {
    render() {
        return <ul className="menubanner">
            <li className="menubanner"><Link to="/quicktranscode">Quick Transcode</Link></li>
            <li className="menubanner last"><Link to="/jobs">Jobs</Link></li>
        </ul>
    }
}

export default MenuBanner;