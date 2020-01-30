import React from 'react';
import PropTypes from 'prop-types';
import {FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import css from './HidableExpander.css';

class HidableExpander extends React.Component {
    static propTypes = {
        headerText: PropTypes.string,
        children: PropTypes.object,
        initialExpanderState: PropTypes.bool
    };

    constructor(props) {
        super(props);

        const initialState = (this.props.initialExpanderState===undefined) ? false : this.props.initialExpanderState;

        this.state = {
            expanded: initialState
        };

        this.expanderClicked = this.expanderClicked.bind(this);
    }

    expanderClicked() {
        this.setState(oldState=>{return {expanded: !oldState.expanded}});
    }

    render() {
        return <div className="hidable-expander">
            <FontAwesomeIcon className="expander-icon" icon={this.state.expanded ? "caret-down" : "caret-right"} onClick={this.expanderClicked}/>
            <span className="hidable-expander-header" onClick={this.expanderClicked}>{this.props.headerText}</span>
            {this.state.expanded ? this.props.children : ""}
        </div>
    }
}

export default HidableExpander;