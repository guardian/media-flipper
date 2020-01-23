import React from 'react';
import {render} from 'react-dom';
import {BrowserRouter, Link, Route, Switch, Redirect, withRouter} from 'react-router-dom';
import { library } from '@fortawesome/fontawesome-svg-core'
import RootComponent from "./RootComponent.jsx";
import css from './approot.css';

class App extends React.Component {
    render() {
        return <Switch>
                <Route path="/" exact={true} component={RootComponent}/>
            </Switch>;
    }
}

const AppWithRouter = withRouter(App);

render(<BrowserRouter root="/"><AppWithRouter/></BrowserRouter>, document.getElementById('app'));

export default App;